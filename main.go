// Command np-update passes PS3 game patch traffic through to Sony: the title patch list
// (/tpl/<env>/<titleid>/<titleid>-ver.xml) and the patch package downloads (/tppkg/...). OpenPak hosts
// no patches. System firmware update hosts never reach this service.
package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"regexp"
	"strings"
	"time"
)

// Sony's patch hosts (a0/b0/gs.ww.np.dl.playstation.net, zeus.dl.playstation.net) are the only upstreams.
const hostSuffix = ".dl.playstation.net"

// defaultHost serves requests that carry no usable Host header.
const defaultHost = "a0.ww.np.dl.playstation.net"

// SCEI DNAS Root 05. Sony's leaf is SHA-1 signed, which Go's verifier refuses, so verifyChain checks it by hand.
//
//go:embed sony_root.pem
var sonyRootPEM []byte

var versionRE = regexp.MustCompile(`<package version="([^"]*)"`)

type hostKey struct{}

func sonyRoot() *x509.Certificate {
	b, _ := pem.Decode(sonyRootPEM)
	if b == nil {
		panic("sony_root.pem")
	}
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		panic(err)
	}
	return c
}

// verifyChain pins the connection to Sony's root: the leaf must be signed (RSA PKCS#1 v1.5, SHA-1) by it,
// be in date, and carry a wildcard or exact common name matching the host.
func verifyChain(root *x509.Certificate, host string, now time.Time, certs []*x509.Certificate) error {
	if len(certs) == 0 {
		return errors.New("no certificate")
	}
	leaf := certs[0]
	if now.Before(leaf.NotBefore) || now.After(leaf.NotAfter) {
		return errors.New("sony leaf certificate out of date")
	}
	cn := leaf.Subject.CommonName
	if !(cn == host || (strings.HasPrefix(cn, "*.") && strings.HasSuffix(host, cn[1:]) && !strings.Contains(strings.TrimSuffix(host, cn[1:]), "."))) {
		return errors.New("sony leaf certificate does not match host")
	}
	if !bytes.Equal(leaf.RawIssuer, root.RawSubject) {
		return errors.New("sony leaf not issued by the pinned root")
	}
	pub, ok := root.PublicKey.(*rsa.PublicKey)
	if !ok || leaf.SignatureAlgorithm != x509.SHA1WithRSA {
		return errors.New("unexpected sony certificate signature")
	}
	sum := sha1.Sum(leaf.RawTBSCertificate)
	return rsa.VerifyPKCS1v15(pub, crypto.SHA1, sum[:], leaf.Signature)
}

func sonyTransport(root *x509.Certificate) *http.Transport {
	return &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		ResponseHeaderTimeout: 20 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		IdleConnTimeout:       60 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // replaced by VerifyConnection below
			VerifyConnection: func(cs tls.ConnectionState) error {
				return verifyChain(root, cs.ServerName, time.Now(), cs.PeerCertificates)
			},
		},
	}
}

func allowedHost(h string) bool {
	return strings.HasSuffix(h, hostSuffix) && !strings.ContainsAny(h, "/@ ")
}

func requestHost(r *http.Request) string {
	h := strings.ToLower(r.Host)
	if i := strings.LastIndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	if h == "" || !strings.Contains(h, ".") || !strings.HasSuffix(h, "playstation.net") {
		return defaultHost
	}
	return h
}

// handler reverse-proxies to Sony. The patch list goes over HTTPS (pinned), packages over HTTP, as Sony's
// own list URLs specify. Anything Sony cannot be reached for answers 404, the "no update" reply VSH accepts.
func handler(tr http.RoundTripper) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			host := pr.In.Context().Value(hostKey{}).(string)
			scheme := "http"
			if strings.HasPrefix(pr.In.URL.Path, "/tpl/") {
				scheme = "https"
			}
			pr.Out.URL.Scheme = scheme
			pr.Out.URL.Host = host
			pr.Out.Host = host
		},
		Transport:     tr,
		FlushInterval: -1,
		ModifyResponse: func(resp *http.Response) error {
			path := resp.Request.URL.Path
			if strings.HasPrefix(path, "/tpl/") && resp.StatusCode == 200 && resp.ContentLength > 0 && resp.ContentLength < 65536 {
				body, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
				resp.Body.Close()
				if err != nil {
					return err
				}
				var versions []string
				for _, m := range versionRE.FindAllSubmatch(body, -1) {
					versions = append(versions, string(m[1]))
				}
				slog.Info("title patch list", "path", path, "status", resp.StatusCode, "versions", versions)
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return nil
			}
			slog.Info("patch passthrough", "path", path, "status", resp.StatusCode, "range", resp.Request.Header.Get("Range"), "bytes", resp.ContentLength)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.Warn("sony upstream unavailable", "path", r.URL.Path, "err", err)
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusNotFound)
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	pass := func(w http.ResponseWriter, r *http.Request) {
		host := requestHost(r)
		if !allowedHost(host) {
			slog.Info("patch host refused", "host", host, "path", r.URL.Path)
			w.Header().Set("Content-Length", "0")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/tppkg/") {
			rc := http.NewResponseController(w)
			rc.SetReadDeadline(time.Now().Add(2 * time.Hour))
			rc.SetWriteDeadline(time.Now().Add(2 * time.Hour))
		}
		proxy.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), hostKey{}, host)))
	}
	mux.HandleFunc("GET /tpl/", pass)
	mux.HandleFunc("GET /tppkg/", pass)
	mux.HandleFunc("HEAD /tppkg/", pass)
	return mux
}

func main() {
	addr := os.Getenv("NP_HTTP_ADDR")
	if strings.TrimSpace(addr) == "" {
		addr = ":8534"
	}
	slog.Info("np-update listening", "addr", addr)
	srv := &http.Server{Addr: addr, Handler: handler(sonyTransport(sonyRoot())), ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("np-update stopped", "err", err)
		os.Exit(1)
	}
}

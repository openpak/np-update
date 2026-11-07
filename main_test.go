package main

import (
	"crypto/x509"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type fake struct{ got []*http.Request }

func (f *fake) RoundTrip(r *http.Request) (*http.Response, error) {
	f.got = append(f.got, r)
	body := `<titlepatch><package version="01.12" size="1"/></titlepatch>`
	return &http.Response{StatusCode: 200, ContentLength: int64(len(body)), Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}, Request: r}, nil
}

type failing struct{}

func (failing) RoundTrip(*http.Request) (*http.Response, error) { return nil, io.ErrUnexpectedEOF }

func TestListGoesToSonyOverHTTPS(t *testing.T) {
	f := &fake{}
	req := httptest.NewRequest("GET", "/tpl/np/NPEB01270/NPEB01270-ver.xml", nil)
	req.Host = "a0.ww.np.dl.playstation.net"
	w := httptest.NewRecorder()
	handler(f).ServeHTTP(w, req)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "01.12") {
		t.Fatalf("%d %q", w.Code, w.Body)
	}
	if u := f.got[0].URL; u.Scheme != "https" || u.Host != "a0.ww.np.dl.playstation.net" {
		t.Fatal(u)
	}
}

func TestPackageGoesOverHTTPAndForeignHostRefused(t *testing.T) {
	f := &fake{}
	h := handler(f)
	req := httptest.NewRequest("GET", "/tppkg/np/X/a.pkg", nil)
	req.Host = "b0.ww.np.dl.playstation.net:80"
	h.ServeHTTP(httptest.NewRecorder(), req)
	if u := f.got[0].URL; u.Scheme != "http" || u.Host != "b0.ww.np.dl.playstation.net" {
		t.Fatal(u)
	}
	for _, host := range []string{"f0101.ps3.update.playstation.net", "evil.playstation.net"} {
		req = httptest.NewRequest("GET", "/tpl/np/X/X-ver.xml", nil)
		req.Host = host
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 404 || len(f.got) != 1 {
			t.Errorf("%s: %d", host, w.Code)
		}
	}
}

func TestUpstreamDownIsNoUpdate(t *testing.T) {
	w := httptest.NewRecorder()
	handler(failing{}).ServeHTTP(w, httptest.NewRequest("GET", "/tpl/np/X/X-ver.xml", nil))
	if w.Code != 404 {
		t.Fatal(w.Code)
	}
}

func TestRootParses(t *testing.T) {
	root := sonyRoot()
	if root.Subject.CommonName != "SCEI DNAS Root 05" {
		t.Fatal(root.Subject)
	}
	if err := verifyChain(root, "a0.ww.np.dl.playstation.net", time.Now(), []*x509.Certificate{root}); err == nil {
		t.Fatal("root must not pass as a leaf")
	}
}

// TestLiveSony fetches a real patch list; set NP_UPDATE_LIVE=1 to run it.
func TestLiveSony(t *testing.T) {
	if os.Getenv("NP_UPDATE_LIVE") == "" {
		t.Skip()
	}
	srv := httptest.NewServer(handler(sonyTransport(sonyRoot())))
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/tpl/np/NPEB01270/NPEB01270-ver.xml", nil)
	req.Host = "a0.ww.np.dl.playstation.net"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !strings.Contains(string(b), "NPEB01270") {
		t.Fatalf("%d %s", resp.StatusCode, b)
	}
}

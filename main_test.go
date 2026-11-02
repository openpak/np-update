package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoUpdateForEveryTitle(t *testing.T) {
	h := handler()
	for _, p := range []string{"/tpl/np/BLUS30001/BLUS30001-ver.xml", "/tpl/np/NPUB30001_00/NPUB30001_00-ver.xml", "/tpl/np/X/X-ver.xml"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != http.StatusNotFound || w.Body.Len() != 0 {
			t.Errorf("%s: %d %q", p, w.Code, w.Body)
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/healthz", nil))
	if w.Code != 200 {
		t.Fatal("health")
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriter_WriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	wrapped := wrapResponseWriter(rr)

	wrapped.WriteHeader(http.StatusNotFound)

	if wrapped.Status() != http.StatusNotFound {
		t.Errorf("Status() = %d, want %d", wrapped.Status(), http.StatusNotFound)
	}
}

func TestResponseWriter_WriteHeaderIdempotent(t *testing.T) {
	rr := httptest.NewRecorder()
	wrapped := wrapResponseWriter(rr)

	wrapped.WriteHeader(http.StatusNotFound)
	wrapped.WriteHeader(http.StatusOK) // should be ignored

	if wrapped.Status() != http.StatusNotFound {
		t.Errorf("Status() = %d, want %d (second WriteHeader should be ignored)", wrapped.Status(), http.StatusNotFound)
	}
}

func TestResponseWriter_DefaultStatus(t *testing.T) {
	rr := httptest.NewRecorder()
	wrapped := wrapResponseWriter(rr)

	if wrapped.Status() != 0 {
		t.Errorf("Status() = %d before any write, want 0", wrapped.Status())
	}
}

func TestLogging(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	handler := Logging(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %d want %d", rr.Code, http.StatusCreated)
	}
}

func TestLogging_DefaultStatus(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	handler := Logging(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %d want %d", rr.Code, http.StatusOK)
	}
}

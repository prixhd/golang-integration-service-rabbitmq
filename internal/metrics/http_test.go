package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMiddleware(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	wrapped := HTTPMiddleware("/test", handler)

	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	recorder := httptest.NewRecorder()

	counter := HTTPRequestsTotal.WithLabelValues("GET", "/test", "201")
	before := testutil.ToFloat64(counter)

	wrapped.ServeHTTP(recorder, request)

	after := testutil.ToFloat64(counter)

	if after != before+1 {
		t.Fatalf("expected counter to increase by 1, before=%v, after=%v", before, after)
	}

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}

}

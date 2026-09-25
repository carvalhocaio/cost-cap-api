package httpx_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/carvalhocaio/cost-cap-api/internal/httpx"
)

type album struct {
	Name string `json:"name"`
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid object", `{"name":"clancy"}`, 0},
		{"empty body", ``, http.StatusBadRequest},
		{"malformed json", `{"name":`, http.StatusBadRequest},
		{"unknown field", `{"name":"clancy","year":2024}`, http.StatusBadRequest},
		{"wrong type", `{"name":2024}`, http.StatusBadRequest},
		{"trailing data", `{"name":"clancy"}{"name":"paladin"}`, http.StatusBadRequest},
		{"oversized body", `{"name":"` + strings.Repeat("x", 1<<20) + `"}`, http.StatusRequestEntityTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))

			got, err := httpx.DecodeJSON[album](httptest.NewRecorder(), request)

			if tt.wantStatus == 0 {
				if err != nil || got.Name != "clancy" {
					t.Fatalf("DecodeJSON() = %+v, %v, want clancy, nil", got, err)
				}
				return
			}

			var httpErr *httpx.Error
			if !errors.As(err, &httpErr) || httpErr.Status != tt.wantStatus {
				t.Fatalf("error = %v, want status %d", err, tt.wantStatus)
			}
		})
	}
}

package incidentrelay

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientValidatesAndNormalizesBaseURL(t *testing.T) {
	client, err := NewClient(ClientConfig{
		BaseURL:   " https://incidentrelay.example.com/ ",
		Token:     " token-value ",
		UserAgent: "test-agent",
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	if got, want := client.baseURL, "https://incidentrelay.example.com"; got != want {
		t.Fatalf("baseURL = %q, want %q", got, want)
	}
	if got, want := client.token, "token-value"; got != want {
		t.Fatalf("token = %q, want %q", got, want)
	}

	if _, err := NewClient(ClientConfig{}); err == nil {
		t.Fatal("NewClient without base_url returned nil error")
	}
}

func TestClientDoSendsJSONAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, http.MethodPost; got != want {
			t.Fatalf("method = %s, want %s", got, want)
		}
		if got, want := r.URL.Path, "/api/test"; got != want {
			t.Fatalf("path = %s, want %s", got, want)
		}
		if got, want := r.Header.Get("Accept"), "application/json"; got != want {
			t.Fatalf("Accept = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Content-Type"), "application/json"; got != want {
			t.Fatalf("Content-Type = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("User-Agent"), "terraform/test terraform-provider-incidentrelay"; got != want {
			t.Fatalf("User-Agent = %q, want %q", got, want)
		}
		if got, want := r.Header.Get("Authorization"), "Bearer test-token"; got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if got, want := body["name"], "primary"; got != want {
			t.Fatalf("body[name] = %v, want %v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true,"id":42}`))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{
		BaseURL:   server.URL + "/",
		Token:     "test-token",
		UserAgent: "terraform/test terraform-provider-incidentrelay",
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	var out map[string]interface{}
	err = client.Do(context.Background(), http.MethodPost, "/api/test", map[string]interface{}{"name": "primary"}, &out)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}

	if got, want := out["ok"], true; got != want {
		t.Fatalf("out[ok] = %v, want %v", got, want)
	}
	if got, want := out["id"], float64(42); got != want {
		t.Fatalf("out[id] = %v, want %v", got, want)
	}
}

func TestClientDoHandlesNotFoundAndErrorBodies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing":
			http.NotFound(w, r)
		case "/boom":
			http.Error(w, "bad news", http.StatusBadGateway)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(ClientConfig{BaseURL: server.URL, Token: "token"})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	if err := client.Do(context.Background(), http.MethodGet, "/missing", nil, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing error = %v, want ErrNotFound", err)
	}

	err = client.Do(context.Background(), http.MethodGet, "/boom", nil, nil)
	if err == nil {
		t.Fatal("boom returned nil error")
	}
	if !strings.Contains(err.Error(), "GET /boom returned 502") || !strings.Contains(err.Error(), "bad news") {
		t.Fatalf("error = %q, want status and response body", err.Error())
	}
}

func TestJoinURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		endpoint string
		want     string
	}{
		{name: "empty endpoint", baseURL: "https://example.com/root", endpoint: "", want: "https://example.com/root"},
		{name: "leading slash", baseURL: "https://example.com/root/", endpoint: "/api/groups", want: "https://example.com/api/groups"},
		{name: "no leading slash", baseURL: "https://example.com/root", endpoint: "api/groups", want: "https://example.com/api/groups"},
		{name: "absolute endpoint", baseURL: "https://example.com/root", endpoint: "https://other.example/api", want: "https://other.example/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := joinURL(tt.baseURL, tt.endpoint)
			if err != nil {
				t.Fatalf("joinURL returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("joinURL = %q, want %q", got, tt.want)
			}
		})
	}
}

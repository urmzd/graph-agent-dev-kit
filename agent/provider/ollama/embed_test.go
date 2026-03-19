package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaEmbedder_Embed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", 404)
			return
		}

		var req EmbedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		// Return a deterministic embedding based on input length.
		vec := make([]float32, 3)
		vec[0] = float32(len(req.Input))
		resp := EmbedResponse{Embeddings: [][]float32{vec}}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-model", "test-embed")
	embedder := NewEmbedder(client)

	results, err := embedder.Embed(context.Background(), []string{"hello", "world!"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0][0] != 5 { // len("hello") = 5
		t.Errorf("results[0][0] = %f, want 5", results[0][0])
	}
	if results[1][0] != 6 { // len("world!") = 6
		t.Errorf("results[1][0] = %f, want 6", results[1][0])
	}
}

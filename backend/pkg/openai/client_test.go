package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRuntimeClientUsesLatestSettings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer new-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("could not decode request: %v", err)
		}
		switch r.URL.Path {
		case "/v1/chat/completions":
			if req.Model != "new-model" {
				t.Errorf("unexpected chat model %q", req.Model)
			}
			_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
		case "/v1/embeddings":
			if req.Model != "embedding-model" {
				t.Errorf("unexpected embedding model %q", req.Model)
			}
			_, _ = fmt.Fprint(w, `{"data":[{"embedding":[0.5]}]}`)
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	var key string
	model := "old-model"
	client := NewRuntimeClient("embedding-model", func() (string, string, string) { return key, srv.URL + "/v1/", model })
	ctx := context.Background()
	if _, err := client.Chat(ctx, nil); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("expected missing key error, got %v", err)
	}
	key = "old-key"
	if _, err := client.Chat(ctx, nil); !errors.Is(err, ErrAuthentication) {
		t.Fatalf("expected authentication error, got %v", err)
	}
	key, model = "new-key", "new-model"
	if text, err := client.Chat(ctx, nil); err != nil || text != "ok" {
		t.Fatalf("expected updated settings to return ok, got %q and %v", text, err)
	}
	if result, err := client.CreateEmbedding(ctx, "example"); err != nil || len(result) != 1 || result[0] != 0.5 {
		t.Fatalf("expected an embedding with the updated credentials, got %v and %v", result, err)
	}
}

func TestExtractImagesFromTaskStatus_NotSucceed(t *testing.T) {
	statuses := []string{"PENDING", "RUNNING", "FAILED", "failed", ""}
	for _, s := range statuses {
		resp := extractImagesFromTaskStatus(&taskStatusResponse{TaskStatus: s})
		if resp != nil {
			t.Errorf("expected nil for status %q, got %+v", s, resp)
		}
	}
}

func TestExtractImagesFromTaskStatus_CaseInsensitive(t *testing.T) {
	cases := []string{"SUCCEED", "Succeed", "succeed", "SUCCEEDED", "succeeded"}
	for _, s := range cases {
		resp := extractImagesFromTaskStatus(&taskStatusResponse{
			TaskStatus:   s,
			OutputImages: []string{"https://example.com/img.png"},
		})
		if resp == nil {
			t.Fatalf("expected non-nil for status %q", s)
		}
		if len(resp.Data) != 1 || resp.Data[0].URL != "https://example.com/img.png" {
			t.Errorf("unexpected data for status %q: %+v", s, resp.Data)
		}
	}
}

func TestExtractImagesFromTaskStatus_OutputResults(t *testing.T) {
	resp := extractImagesFromTaskStatus(&taskStatusResponse{
		TaskStatus: "SUCCEED",
		Output: &struct {
			TaskID  string          `json:"task_id"`
			Results []ImageDataItem `json:"results"`
		}{
			Results: []ImageDataItem{
				{URL: "https://example.com/a.png"},
				{B64JSON: "base64data"},
			},
		},
	})
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.Data))
	}
	if resp.Data[0].URL != "https://example.com/a.png" {
		t.Errorf("unexpected URL: %s", resp.Data[0].URL)
	}
	if resp.Data[1].B64JSON != "base64data" {
		t.Errorf("unexpected B64JSON: %s", resp.Data[1].B64JSON)
	}
}

func TestExtractImagesFromTaskStatus_SucceedNoImages(t *testing.T) {
	resp := extractImagesFromTaskStatus(&taskStatusResponse{TaskStatus: "SUCCEED"})
	if resp != nil {
		t.Errorf("expected nil when SUCCEED but no images, got %+v", resp)
	}
}

func TestHandlePollStatus_Error(t *testing.T) {
	_, err := handlePollStatus(&taskStatusResponse{
		Error: &struct {
			Message string `json:"message"`
		}{Message: "out of quota"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "image task failed: out of quota" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestHandlePollStatus_SucceedWithImages(t *testing.T) {
	resp, err := handlePollStatus(&taskStatusResponse{
		TaskStatus:   "SUCCEED",
		OutputImages: []string{"https://example.com/img.png"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %+v", resp)
	}
}

func TestHandlePollStatus_SucceedNoImages(t *testing.T) {
	_, err := handlePollStatus(&taskStatusResponse{TaskStatus: "SUCCEEDED"})
	if err == nil {
		t.Fatal("expected error for succeeded with no results")
	}
}

func TestHandlePollStatus_Failed(t *testing.T) {
	_, err := handlePollStatus(&taskStatusResponse{
		TaskStatus: "failed",
		Message:    "content policy violation",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "content policy violation" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestHandlePollStatus_FailedNoMessage(t *testing.T) {
	_, err := handlePollStatus(&taskStatusResponse{TaskStatus: "FAILED"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got != "image generation task failed" {
		t.Errorf("unexpected error: %s", got)
	}
}

func TestHandlePollStatus_Pending(t *testing.T) {
	resp, err := handlePollStatus(&taskStatusResponse{TaskStatus: "PENDING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response for PENDING, got %+v", resp)
	}
}

func TestHandlePollStatus_Running(t *testing.T) {
	resp, err := handlePollStatus(&taskStatusResponse{TaskStatus: "RUNNING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response for RUNNING, got %+v", resp)
	}
}

func TestNewClient_DefaultBaseURL(t *testing.T) {
	c := NewClient("key", "", "model", "dummy-embedding")
	if c.baseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default base URL, got %s", c.baseURL)
	}
	if c.apiKey != "key" {
		t.Errorf("expected apiKey=key, got %s", c.apiKey)
	}
	if c.model != "model" {
		t.Errorf("expected model=model, got %s", c.model)
	}
}

func TestNewClient_CustomBaseURL(t *testing.T) {
	c := NewClient("k", "https://custom.api/v1", "m", "dummy-embedding")
	if c.baseURL != "https://custom.api/v1" {
		t.Errorf("expected custom base URL, got %s", c.baseURL)
	}
}

func TestHandleAsyncTask_InvalidJSON(t *testing.T) {
	c := NewClient("key", "http://localhost", "model", "dummy-embedding")
	resp, err := c.handleAsyncTask(context.Background(), []byte("not json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response for invalid JSON, got %+v", resp)
	}
}

func TestHandleAsyncTask_AlreadySucceed(t *testing.T) {
	c := NewClient("key", "http://localhost", "model", "dummy-embedding")
	body, _ := json.Marshal(taskStatusResponse{
		TaskStatus:   "SUCCEED",
		OutputImages: []string{"https://example.com/img.png"},
	})
	resp, err := c.handleAsyncTask(context.Background(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %+v", resp)
	}
	if resp.Data[0].URL != "https://example.com/img.png" {
		t.Errorf("unexpected URL: %s", resp.Data[0].URL)
	}
}

func TestHandleAsyncTask_NoTaskIDOrRequestID(t *testing.T) {
	c := NewClient("key", "http://localhost", "model", "dummy-embedding")
	body, _ := json.Marshal(taskStatusResponse{
		TaskStatus: "PENDING",
	})
	resp, err := c.handleAsyncTask(context.Background(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil for PENDING with no IDs, got %+v", resp)
	}
}

func TestHandleAsyncTask_PollWithTaskID(t *testing.T) {
	// Set up test server that returns SUCCEED on first poll
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(taskStatusResponse{
			TaskStatus:   "SUCCEED",
			OutputImages: []string{"https://example.com/polled.png"},
		})
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "dummy-embedding")
	body, _ := json.Marshal(taskStatusResponse{
		TaskID:     "task-123",
		TaskStatus: "RUNNING",
	})
	resp, err := c.handleAsyncTask(context.Background(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Data) != 1 {
		t.Fatalf("expected 1 image from poll, got %+v", resp)
	}
	if resp.Data[0].URL != "https://example.com/polled.png" {
		t.Errorf("unexpected URL: %s", resp.Data[0].URL)
	}
}

func TestHandleAsyncTask_PollWithRequestID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(taskStatusResponse{
			TaskStatus:   "SUCCEED",
			OutputImages: []string{"https://example.com/fallback.png"},
		})
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "dummy-embedding")
	body, _ := json.Marshal(taskStatusResponse{
		RequestID:  "req-456",
		TaskStatus: "PENDING",
	})
	resp, err := c.handleAsyncTask(context.Background(), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Data) != 1 {
		t.Fatalf("expected 1 image from poll, got %+v", resp)
	}
	if resp.Data[0].URL != "https://example.com/fallback.png" {
		t.Errorf("unexpected URL: %s", resp.Data[0].URL)
	}
}

func TestHandleAsyncTask_PollError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(taskStatusResponse{
			Error: &struct {
				Message string `json:"message"`
			}{Message: "quota exceeded"},
		})
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "dummy-embedding")
	body, _ := json.Marshal(taskStatusResponse{
		TaskID:     "task-err",
		TaskStatus: "RUNNING",
	})
	_, err := c.handleAsyncTask(context.Background(), body)
	if err == nil {
		t.Fatal("expected error from poll")
	}
}

func TestChat_InvalidURL(t *testing.T) {
	c := NewClient("key", "http://invalid.localhost:0", "model", "dummy-embedding")
	_, err := c.Chat(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestChat_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "internal error")
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "dummy-embedding")
	_, err := c.Chat(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestChat_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "hi there",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "dummy-embedding")
	result, err := c.Chat(context.Background(), []Message{
		{Role: "user", Content: "hello"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "hi there" {
		t.Errorf("expected 'hi there', got %q", result)
	}
}

func TestGenerateImage_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, "bad request")
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "dummy-embedding")
	_, err := c.GenerateImage(context.Background(), "model", "a cat")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestGenerateImage_AcceptedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, `{"task_status":"SUCCEED","output_images":["https://example.test/cover.png"]}`)
	}))
	defer srv.Close()

	client := NewClient("key", srv.URL, "model", "embed-model")
	image, err := client.GenerateImage(context.Background(), "image-model", "a cover")
	if err != nil || image == nil || len(image.Data) != 1 || image.Data[0].URL != "https://example.test/cover.png" {
		t.Fatalf("unexpected accepted response: %+v, %v", image, err)
	}
}

func TestGenerateImage_SyncResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := ImageResponse{
			Data: []ImageDataItem{
				{URL: "https://example.com/generated.png"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "dummy-embedding")
	resp, err := c.GenerateImage(context.Background(), "model", "a cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.Data) != 1 {
		t.Fatalf("expected 1 image, got %+v", resp)
	}
	if resp.Data[0].URL != "https://example.com/generated.png" {
		t.Errorf("unexpected URL: %s", resp.Data[0].URL)
	}
}

func TestCreateEmbedding_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"embedding": []float32{0.1, 0.2, 0.3}},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "embed-model")
	emb, err := c.CreateEmbedding(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(emb) != 3 {
		t.Fatalf("expected 3 floats, got %d", len(emb))
	}
}

func TestCreateEmbedding_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "embed-model")
	_, err := c.CreateEmbedding(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestCreateEmbedding_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "embed-model")
	_, err := c.CreateEmbedding(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error for empty data")
	}
}

func TestDoPost_SetsHeaders(t *testing.T) {
	var gotAuth, gotContentType string
	var gotCustom string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotCustom = r.Header.Get("X-Custom")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient("test-key", srv.URL, "m", "e")
	_, _ = c.doPost(context.Background(), "/test", map[string]string{"a": "b"}, 1024, map[string]string{"X-Custom": "val"})

	if gotAuth != "Bearer test-key" {
		t.Errorf("auth = %q, want 'Bearer test-key'", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("content-type = %q", gotContentType)
	}
	if gotCustom != "val" {
		t.Errorf("custom header = %q, want 'val'", gotCustom)
	}
}

// --- parseRerankScores tests ---

func TestParseRerankScores_DirectJSON(t *testing.T) {
	scores, err := parseRerankScores(`[{"index":0,"score":8},{"index":1,"score":3}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}
	if scores[0].Index != 0 || scores[0].Score != 8 {
		t.Errorf("score[0] = %+v", scores[0])
	}
	if scores[1].Index != 1 || scores[1].Score != 3 {
		t.Errorf("score[1] = %+v", scores[1])
	}
}

func TestParseRerankScores_MarkdownWrapped(t *testing.T) {
	input := "```json\n[{\"index\":0,\"score\":7}]\n```"
	scores, err := parseRerankScores(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scores) != 1 || scores[0].Score != 7 {
		t.Errorf("unexpected scores: %+v", scores)
	}
}

func TestParseRerankScores_WithPreamble(t *testing.T) {
	input := "Here are the scores:\n[{\"index\":0,\"score\":5},{\"index\":1,\"score\":9}]\nDone."
	scores, err := parseRerankScores(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}
}

func TestParseRerankScores_NoArray(t *testing.T) {
	_, err := parseRerankScores("no json here at all")
	if err == nil {
		t.Fatal("expected error for input without JSON array")
	}
}

func TestParseRerankScores_EmptyArray(t *testing.T) {
	scores, err := parseRerankScores("[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(scores) != 0 {
		t.Errorf("expected empty array, got %d", len(scores))
	}
}

func TestParseRerankScores_InvalidJSON(t *testing.T) {
	_, err := parseRerankScores("[{bad json}]")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- Rerank integration tests ---

func TestRerank_EmptyCandidates(t *testing.T) {
	c := NewClient("key", "http://localhost", "model", "embed")
	results, err := c.Rerank(context.Background(), "query", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil, got %+v", results)
	}
}

func TestRerank_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `[{"index":0,"score":8},{"index":1,"score":2}]`,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "embed")
	candidates := []RerankCandidate{
		{ID: "a", Content: "doc1"},
		{ID: "b", Content: "doc2"},
	}
	results, err := c.Rerank(context.Background(), "test query", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != "a" || results[0].Score != 8 {
		t.Errorf("results[0] = %+v", results[0])
	}
	if results[1].ID != "b" || results[1].Score != 2 {
		t.Errorf("results[1] = %+v", results[1])
	}
}

func TestRerank_OutOfBoundsIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": `[{"index":0,"score":5},{"index":99,"score":10}]`,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "embed")
	candidates := []RerankCandidate{{ID: "a", Content: "doc1"}}
	results, err := c.Rerank(context.Background(), "query", candidates)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Score != 5 {
		t.Errorf("expected score 5, got %f", results[0].Score)
	}
}

func TestRerank_LLMError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model", "embed")
	candidates := []RerankCandidate{{ID: "a", Content: "doc1"}}
	_, err := c.Rerank(context.Background(), "query", candidates)
	if err == nil {
		t.Fatal("expected error for server failure")
	}
}

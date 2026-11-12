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
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer new-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model != "new-model" {
			t.Errorf("unexpected request model %q or decode error %v", req.Model, err)
		}
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer srv.Close()

	var key string
	model := "old-model"
	client := NewRuntimeClient(func() (string, string, string) { return key, srv.URL + "/v1/", model })
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
			TaskID  string `json:"task_id"`
			Results []struct {
				URL     string `json:"url,omitempty"`
				B64JSON string `json:"b64_json,omitempty"`
			} `json:"results"`
		}{
			Results: []struct {
				URL     string `json:"url,omitempty"`
				B64JSON string `json:"b64_json,omitempty"`
			}{
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
	c := NewClient("key", "", "model")
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
	c := NewClient("k", "https://custom.api/v1", "m")
	if c.baseURL != "https://custom.api/v1" {
		t.Errorf("expected custom base URL, got %s", c.baseURL)
	}
}

func TestHandleAsyncTask_InvalidJSON(t *testing.T) {
	c := NewClient("key", "http://localhost", "model")
	resp, err := c.handleAsyncTask(context.Background(), []byte("not json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != nil {
		t.Errorf("expected nil response for invalid JSON, got %+v", resp)
	}
}

func TestHandleAsyncTask_AlreadySucceed(t *testing.T) {
	c := NewClient("key", "http://localhost", "model")
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
	c := NewClient("key", "http://localhost", "model")
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

	c := NewClient("key", srv.URL, "model")
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

	c := NewClient("key", srv.URL, "model")
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

	c := NewClient("key", srv.URL, "model")
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
	c := NewClient("key", "http://invalid.localhost:0", "model")
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

	c := NewClient("key", srv.URL, "model")
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

	c := NewClient("key", srv.URL, "model")
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

	c := NewClient("key", srv.URL, "model")
	_, err := c.GenerateImage(context.Background(), "model", "a cat")
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestGenerateImage_SyncResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := ImageResponse{
			Data: []struct {
				URL     string `json:"url,omitempty"`
				B64JSON string `json:"b64_json,omitempty"`
			}{
				{URL: "https://example.com/generated.png"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := NewClient("key", srv.URL, "model")
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

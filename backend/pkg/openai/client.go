package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	maxChatResponseSize  = 1 << 20  // 1 MB
	maxImageResponseSize = 10 << 20 // 10 MB
	headerAuthorization  = "Authorization"
	bearerPrefix         = "Bearer "
)

var (
	ErrNotConfigured      = errors.New("AI 服务尚未配置，请联系管理员设置 OPENAI_API_KEY")
	ErrAuthentication     = errors.New("AI 服务认证失败，请联系管理员更新 OPENAI_API_KEY")
	ErrImageNotConfigured = errors.New("尚未配置 AI 图片生成")
)

type Client struct {
	apiKey          string
	baseURL         string
	model           string
	embeddingModel  string
	http            *http.Client
	runtimeSettings func() (apiKey, baseURL, model string)
}

func NewClient(apiKey, baseURL, model, embeddingModel string) *Client {
	return &Client{
		apiKey:         strings.TrimSpace(apiKey),
		baseURL:        normalizeBaseURL(baseURL),
		model:          model,
		embeddingModel: embeddingModel,
		http:           &http.Client{Timeout: 60 * time.Second},
	}
}

func NewRuntimeClient(embeddingModel string, settings func() (apiKey, baseURL, model string)) *Client {
	c := NewClient("", "", "", embeddingModel)
	c.runtimeSettings = settings
	return c
}

func normalizeBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return "https://api.openai.com/v1"
	}
	return baseURL
}

func (c *Client) snapshot() *Client {
	if c.runtimeSettings == nil {
		return c
	}
	key, baseURL, model := c.runtimeSettings()
	return &Client{
		apiKey: strings.TrimSpace(key), baseURL: normalizeBaseURL(baseURL),
		model: model, embeddingModel: c.embeddingModel, http: c.http,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.snapshot().apiKey != ""
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string    `json:"model"`
	Messages       []Message `json:"messages"`
	Temperature    float64   `json:"temperature"`
	MaxTokens      int       `json:"max_tokens"`
	EnableThinking bool      `json:"enable_thinking"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// doPost sends an authenticated POST request and returns the response body.
// It handles marshaling, header setup, and status code validation.
func (c *Client) doPost(ctx context.Context, path string, reqBody any, maxSize int64, extraHeaders map[string]string) ([]byte, error) {
	if c.apiKey == "" {
		return nil, ErrNotConfigured
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerAuthorization, bearerPrefix+c.apiKey)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to %s failed: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if isAuthenticationFailure(resp.StatusCode) {
		return nil, ErrAuthentication
	}
	acceptedAsyncImage := path == "/images/generations" && resp.StatusCode == http.StatusAccepted
	if resp.StatusCode != http.StatusOK && !acceptedAsyncImage {
		return nil, fmt.Errorf("request to %s failed: status %d", path, resp.StatusCode)
	}

	return respBody, nil
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	c = c.snapshot()
	reqBody := chatRequest{
		Model:          c.model,
		Messages:       messages,
		Temperature:    0.7,
		MaxTokens:      4096,
		EnableThinking: false,
	}

	respBody, err := c.doPost(ctx, "/chat/completions", reqBody, maxChatResponseSize, nil)
	if err != nil {
		return "", err
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("openai error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Client) CreateEmbedding(ctx context.Context, input string) ([]float32, error) {
	c = c.snapshot()
	reqBody := embeddingRequest{
		Model: c.embeddingModel,
		Input: input,
	}

	respBody, err := c.doPost(ctx, "/embeddings", reqBody, maxChatResponseSize, nil)
	if err != nil {
		return nil, err
	}

	var embResp embeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if embResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", embResp.Error.Message)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("openai returned no embeddings")
	}

	return embResp.Data[0].Embedding, nil
}

// RerankCandidate is a single candidate for LLM-based reranking.
type RerankCandidate struct {
	ID      string
	Content string
}

// RerankResult holds the reranked score for a candidate.
type RerankResult struct {
	ID    string
	Score float64
}

// rerankScoreItem represents a single score from the rerank LLM response.
type rerankScoreItem struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

// parseRerankScores parses the JSON array of scores from raw LLM output.
func parseRerankScores(raw string) ([]rerankScoreItem, error) {
	raw = strings.TrimSpace(raw)
	var scores []rerankScoreItem
	if err := json.Unmarshal([]byte(raw), &scores); err == nil {
		return scores, nil
	}
	// Try extracting JSON array from markdown wrapper
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(raw[start:end+1]), &scores); err == nil {
			return scores, nil
		}
		return nil, fmt.Errorf("failed to parse rerank JSON array: %s", raw[start:end+1])
	}
	return nil, fmt.Errorf("no JSON array found in rerank response")
}

// Rerank uses the LLM to score each candidate's relevance to the query.
// Returns scores in the same order as candidates. On failure, returns nil error
// so callers can fall back to the original ranking.
func (c *Client) Rerank(ctx context.Context, query string, candidates []RerankCandidate) ([]RerankResult, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	for i, cand := range candidates {
		fmt.Fprintf(&sb, "[%d] %s\n", i, cand.Content)
	}

	prompt := fmt.Sprintf(`给定用户问题和候选文档列表，为每个文档的相关性打分（0-10分，10分最相关）。
只返回JSON数组，格式：[{"index":0,"score":8},{"index":1,"score":3}]
不要返回其他内容。

用户问题：%s

候选文档：
%s`, query, sb.String())

	messages := []Message{
		{Role: "system", Content: "你是一个文档相关性评分助手。只返回JSON数组。"},
		{Role: "user", Content: prompt},
	}

	rawContent, err := c.Chat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("rerank LLM call failed: %w", err)
	}

	scores, err := parseRerankScores(rawContent)
	if err != nil {
		return nil, err
	}

	results := make([]RerankResult, len(candidates))
	for i := range candidates {
		results[i] = RerankResult{ID: candidates[i].ID, Score: 0}
	}
	for _, s := range scores {
		if s.Index >= 0 && s.Index < len(results) {
			results[s.Index].Score = s.Score
		}
	}

	return results, nil
}

type imageRequest struct {
	Model             string  `json:"model"`
	Prompt            string  `json:"prompt"`
	N                 int     `json:"n,omitempty"`
	Size              string  `json:"size,omitempty"`
	Width             int     `json:"width,omitempty"`
	Height            int     `json:"height,omitempty"`
	NumInferenceSteps int     `json:"num_inference_steps,omitempty"`
	GuidanceScale     float64 `json:"guidance_scale"`
}

// ImageDataItem represents a single image result.
type ImageDataItem struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
}

type ImageResponse struct {
	Data  []ImageDataItem `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// handleAsyncTask detects and handles async task responses from ModelScope.
// Returns an ImageResponse if the task completed synchronously or after polling.
// Returns nil, nil if the response is not an async task.
func (c *Client) handleAsyncTask(ctx context.Context, respBody []byte) (*ImageResponse, error) {
	var taskResp taskStatusResponse
	if err := json.Unmarshal(respBody, &taskResp); err != nil {
		return nil, nil
	}
	// If already SUCCEED with images, return directly
	if imgResp := extractImagesFromTaskStatus(&taskResp); imgResp != nil {
		return imgResp, nil
	}
	// Has task_id - poll with it (ModelScope uses task_id, NOT request_id)
	if taskResp.TaskID != "" {
		log.Printf("[ImageGen] async task, task_id=%s, status=%s, polling...", taskResp.TaskID, taskResp.TaskStatus)
		return c.pollImageTask(ctx, taskResp.TaskID)
	}
	// Fallback: try request_id if no task_id
	if taskResp.RequestID != "" {
		log.Printf("[ImageGen] async task (fallback), request_id=%s, status=%s, polling...", taskResp.RequestID, taskResp.TaskStatus)
		return c.pollImageTask(ctx, taskResp.RequestID)
	}
	return nil, nil
}

func (c *Client) GenerateImage(ctx context.Context, model, prompt string) (*ImageResponse, error) {
	c = c.snapshot()
	reqBody := imageRequest{
		Model:             model,
		Prompt:            prompt,
		Width:             1024,
		Height:            1024,
		NumInferenceSteps: 9,
		GuidanceScale:     0.0,
	}

	headers := map[string]string{"X-ModelScope-Async-Mode": "true"}
	respBody, err := c.doPost(ctx, "/images/generations", reqBody, maxImageResponseSize, headers)
	if err != nil {
		return nil, err
	}
	log.Printf("[ImageGen] model=%s POST body_len=%d", model, len(respBody))

	// Try parsing as an async task response (ModelScope returns task_id for polling)
	if asyncResp, asyncErr := c.handleAsyncTask(ctx, respBody); asyncResp != nil || asyncErr != nil {
		return asyncResp, asyncErr
	}

	// Synchronous response (standard OpenAI format)
	var imgResp ImageResponse
	if err := json.Unmarshal(respBody, &imgResp); err != nil {
		return nil, fmt.Errorf("failed to parse image response: %w", err)
	}

	if imgResp.Error != nil {
		return nil, fmt.Errorf("openai image error: %s", imgResp.Error.Message)
	}

	if len(imgResp.Data) == 0 {
		return nil, fmt.Errorf("image generation returned no results")
	}

	return &imgResp, nil
}

func extractImagesFromTaskStatus(status *taskStatusResponse) *ImageResponse {
	upper := strings.ToUpper(status.TaskStatus)
	if upper != "SUCCEED" && upper != "SUCCEEDED" {
		return nil
	}

	if len(status.OutputImages) > 0 {
		data := make([]ImageDataItem, len(status.OutputImages))
		for i, u := range status.OutputImages {
			data[i].URL = u
		}
		return &ImageResponse{Data: data}
	}

	if status.Output != nil && len(status.Output.Results) > 0 {
		data := make([]ImageDataItem, len(status.Output.Results))
		for i, r := range status.Output.Results {
			data[i].URL = r.URL
			data[i].B64JSON = r.B64JSON
		}
		return &ImageResponse{Data: data}
	}

	return nil // Status is SUCCEED but no images found
}

type taskStatusResponse struct {
	RequestID    string   `json:"request_id"`
	TaskID       string   `json:"task_id"`
	TaskStatus   string   `json:"task_status"`
	OutputImages []string `json:"output_images,omitempty"`
	Output       *struct {
		TaskID  string          `json:"task_id"`
		Results []ImageDataItem `json:"results"`
	} `json:"output,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// handlePollStatus evaluates a poll response and returns a result, an error, or (nil, nil) to continue polling.
func handlePollStatus(status *taskStatusResponse) (*ImageResponse, error) {
	if status.Error != nil {
		return nil, fmt.Errorf("image task failed: %s", status.Error.Message)
	}
	if imgResp := extractImagesFromTaskStatus(status); imgResp != nil {
		return imgResp, nil
	}
	switch strings.ToUpper(status.TaskStatus) {
	case "SUCCEED", "SUCCEEDED":
		return nil, fmt.Errorf("image task succeeded but no results in response")
	case "FAILED":
		msg := status.Message
		if msg == "" {
			msg = "image generation task failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	// PENDING, RUNNING - continue polling
	return nil, nil
}

func (c *Client) pollImageTask(ctx context.Context, taskID string) (*ImageResponse, error) {
	pollURL := c.baseURL + "/tasks/" + taskID

	for i := 0; i < 40; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
		}

		req, err := http.NewRequestWithContext(ctx, "GET", pollURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create poll request: %w", err)
		}
		req.Header.Set(headerAuthorization, bearerPrefix+c.apiKey)
		req.Header.Set("X-ModelScope-Task-Type", "image_generation")

		resp, err := c.http.Do(req)
		if err != nil {
			log.Printf("[ImageGen] poll error: %v", err)
			continue
		}

		respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxImageResponseSize))
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		log.Printf("[ImageGen] poll #%d status=%d body_len=%d", i+1, resp.StatusCode, len(respBody))
		if isAuthenticationFailure(resp.StatusCode) {
			return nil, ErrAuthentication
		}

		var status taskStatusResponse
		if err := json.Unmarshal(respBody, &status); err != nil {
			log.Printf("[ImageGen] poll parse error: %v", err)
			continue
		}

		if imgResp, err := handlePollStatus(&status); imgResp != nil || err != nil {
			return imgResp, err
		}
	}

	return nil, fmt.Errorf("image generation timed out after polling")
}

func isAuthenticationFailure(status int) bool {
	return status == http.StatusUnauthorized || status == http.StatusForbidden
}

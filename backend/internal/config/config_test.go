package config

import (
	"os"
	"testing"
)

func TestGetEnvFloat64_Default(t *testing.T) {
	result := getEnvFloat64("TEST_FLOAT64_NONEXISTENT_KEY", 0.75)
	if result != 0.75 {
		t.Errorf("expected 0.75, got %f", result)
	}
}

func TestGetEnvFloat64_Valid(t *testing.T) {
	t.Setenv("TEST_FLOAT64_VALID", "0.95")

	result := getEnvFloat64("TEST_FLOAT64_VALID", 0.5)
	if result != 0.95 {
		t.Errorf("expected 0.95, got %f", result)
	}
}

func TestGetEnvFloat64_Invalid(t *testing.T) {
	t.Setenv("TEST_FLOAT64_INVALID", "not-a-number")

	result := getEnvFloat64("TEST_FLOAT64_INVALID", 0.8)
	if result != 0.8 {
		t.Errorf("expected fallback 0.8, got %f", result)
	}
}

func TestGetEnvBool_Default(t *testing.T) {
	result := getEnvBool("TEST_BOOL_NONEXISTENT_KEY", true)
	if !result {
		t.Error("expected true")
	}
}

func TestGetEnvBool_TrueValues(t *testing.T) {
	for _, val := range []string{"true", "1", "yes", "TRUE", "Yes"} {
		t.Setenv("TEST_BOOL_TRUE", val)
		result := getEnvBool("TEST_BOOL_TRUE", false)
		if !result {
			t.Errorf("expected true for %q", val)
		}
	}
}

func TestGetEnvBool_FalseValues(t *testing.T) {
	for _, val := range []string{"false", "0", "no", "FALSE", "No"} {
		t.Setenv("TEST_BOOL_FALSE", val)
		result := getEnvBool("TEST_BOOL_FALSE", true)
		if result {
			t.Errorf("expected false for %q", val)
		}
	}
}

func TestGetEnvBool_InvalidFallback(t *testing.T) {
	t.Setenv("TEST_BOOL_INVALID", "maybe")

	result := getEnvBool("TEST_BOOL_INVALID", true)
	if !result {
		t.Error("expected fallback true for invalid value")
	}
}

func TestLoad_RAGDefaults(t *testing.T) {
	cfg := Load()
	if cfg.RAGSimilarityThreshold != 0.8 {
		t.Errorf("expected default threshold 0.8, got %f", cfg.RAGSimilarityThreshold)
	}
	if cfg.RAGTopK != 5 {
		t.Errorf("expected default top-k 5, got %d", cfg.RAGTopK)
	}
	if !cfg.RAGRerankEnabled {
		t.Error("expected default rerank enabled true")
	}
}

func TestLoad_RAGCustom(t *testing.T) {
	t.Setenv("RAG_SIMILARITY_THRESHOLD", "0.6")
	t.Setenv("RAG_TOP_K", "10")
	t.Setenv("RAG_RERANK_ENABLED", "false")

	cfg := Load()
	if cfg.RAGSimilarityThreshold != 0.6 {
		t.Errorf("expected 0.6, got %f", cfg.RAGSimilarityThreshold)
	}
	if cfg.RAGTopK != 10 {
		t.Errorf("expected 10, got %d", cfg.RAGTopK)
	}
	if cfg.RAGRerankEnabled {
		t.Error("expected false")
	}
}

func TestLoadImageModel(t *testing.T) {
	t.Setenv("JWT_SECRET_KEY", "test-secret")
	t.Setenv("IMAGE_MODEL", "")
	if got := Load().ImageModel; got != "" {
		t.Fatalf("an explicitly blank IMAGE_MODEL should disable image generation, got %q", got)
	}

	if err := os.Unsetenv("IMAGE_MODEL"); err != nil {
		t.Fatal(err)
	}
	if got := Load().ImageModel; got != "Tongyi-MAI/Z-Image-Turbo" {
		t.Fatalf("an unset IMAGE_MODEL should preserve the default, got %q", got)
	}
}

func TestLoadBlankEmbeddingModel(t *testing.T) {
	t.Setenv("EMBEDDING_MODEL", "")
	if got := Load().EmbeddingModel; got != "" {
		t.Fatalf("an explicitly blank EMBEDDING_MODEL should disable RAG, got %q", got)
	}
}

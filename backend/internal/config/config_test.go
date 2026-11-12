package config

import (
	"os"
	"testing"
)

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

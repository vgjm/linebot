package gemini

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestGenerateContent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*27)
	defer cancel()
	apiKey := os.Getenv("GEMINI_API_KEY")
	g, err := New(ctx, apiKey, "")
	if err != nil {
		t.Errorf("failed to create client: %v", err)
	}
	if _, err := g.GenerateContent(ctx, "You are an assistant", "Hello"); err != nil {
		t.Errorf("failed to generate response: %v", err)
	}
}

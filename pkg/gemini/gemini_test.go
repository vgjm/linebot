package gemini

import (
	"context"
	"testing"
	"time"
)

func TestGenerateResponse(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*27)
	defer cancel()
	g, err := New(ctx)
	if err != nil {
		t.Errorf("failed to create client: %v", err)
	}
	if _, err := g.GenerateResponse(ctx, "Hello"); err != nil {
		t.Errorf("failed to generate response: %v", err)
	}
}

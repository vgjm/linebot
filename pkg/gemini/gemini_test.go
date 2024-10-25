package gemini

import (
	"context"
	"testing"
)

func TestGenerateResponse(t *testing.T) {
	ctx := context.Background()
	g, err := New(ctx)
	if err != nil {
		t.Errorf("failed to create client: %v", err)
	}
	if _, err := g.GenerateResponse("Hello"); err != nil {
		t.Errorf("failed to generate response: %v", err)
	}
}

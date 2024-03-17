package geminiclient

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	ctx    context.Context
	client *genai.Client
}

func New(apiKey string) (*GeminiClient, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &GeminiClient{ctx, client}, nil
}

func (gc *GeminiClient) SingleQuestion(ask string) (string, error) {
	model := gc.client.GenerativeModel("gemini-pro")
	model.SetMaxOutputTokens(100)
	resp, err := model.GenerateContent(gc.ctx, genai.Text(ask))
	if err != nil {
		switch err.(type) {
		case *genai.BlockedError:
			return "底线！", nil
		default:
			return "", err
		}
	}
	var text string
	for _, v := range resp.Candidates {
		for _, u := range v.Content.Parts {
			text += fmt.Sprintf("%v", u)
		}
	}
	return text, nil
}
func (gc *GeminiClient) Close() error {
	return gc.client.Close()
}

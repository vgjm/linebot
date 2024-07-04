package geminiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiClient struct {
	ctx    context.Context
	client *genai.Client
	cfg    Configuration
}

type Configuration struct {
	Prompts []Prompt
}

type Prompt struct {
	Text string
	Role string
}

func New(apiKey string, prompts string) (*GeminiClient, error) {
	promptsStr := os.Getenv(prompts)
	var cfg Configuration
	err := json.Unmarshal([]byte(promptsStr), &cfg)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &GeminiClient{ctx, client, cfg}, nil
}

func (gc *GeminiClient) Generate(question string) (string, error) {
	model := gc.client.GenerativeModel("gemini-1.5-flash")

	// Set all harm block to none
	// https://ai.google.dev/docs/safety_setting_gemini?hl=zh-cn#safety-settings
	model.SafetySettings = []*genai.SafetySetting{
		{
			Category:  genai.HarmCategoryHarassment,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryHateSpeech,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategorySexuallyExplicit,
			Threshold: genai.HarmBlockNone,
		},
		{
			Category:  genai.HarmCategoryDangerousContent,
			Threshold: genai.HarmBlockNone,
		},
	}

	// Initial a bot with specified role
	cs := model.StartChat()
	contents := make([]*genai.Content, 0, 10)
	for _, prompt := range gc.cfg.Prompts {
		contents = append(contents, &genai.Content{
			Parts: []genai.Part{
				genai.Text(prompt.Text),
			},
			Role: prompt.Role,
		})
	}
	cs.History = contents

	resp, err := cs.SendMessage(gc.ctx, genai.Text(question))
	if err != nil {
		return "Gemini已麻。", err
	}

	var text string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				text += fmt.Sprintln(part)
			}
		}
	}
	if text == "" {
		switch resp.PromptFeedback.BlockReason {
		case genai.BlockReasonUnspecified:
			text = "Blocked with reason unspecified."
		case genai.BlockReasonSafety:
			text = "Blocked with reason safety."
		case genai.BlockReasonOther:
			text = "Blocked with reason other."
		}
	}

	return text, nil
}

func (gc *GeminiClient) Close() error {
	return gc.client.Close()
}

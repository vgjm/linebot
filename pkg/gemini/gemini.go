package gemini

import (
	"context"
	"fmt"

	"github.com/vgjm/linebot/pkg/llm"
	"google.golang.org/genai"
)

var _ llm.LLM = (*Gemini)(nil)

var DefaultModels = []string{"gemini-2.5-flash", "gemini-2.5-flash-lite", "gemini-2.0-flash-lite"}

type Gemini struct {
	ctx    context.Context
	client *genai.Client
	models []string
}

func New(ctx context.Context, apiKey string, model string) (*Gemini, error) {
	models := DefaultModels
	if model != "" {
		models = append([]string{model}, models...)
	}
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &Gemini{
		ctx:    ctx,
		client: client,
		models: models,
	}, nil
}

func (g *Gemini) GenerateContent(ctx context.Context, instruction, question string) (string, error) {
	config := &genai.GenerateContentConfig{
		// Set all harm block to none
		// https://ai.google.dev/docs/safety_setting_gemini?hl=zh-cn#safety-settings
		SafetySettings: []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryHarassment,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
			{
				Category:  genai.HarmCategoryHateSpeech,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
			{
				Category:  genai.HarmCategorySexuallyExplicit,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockThresholdBlockNone,
			},
		},
	}

	if instruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{
				Text: instruction,
			}},
		}
	}

	var resp *genai.GenerateContentResponse
	var err error
	for _, m := range g.models {
		resp, err = g.client.Models.GenerateContent(ctx, m, genai.Text(question), config)
		if err != nil {
			continue
		}

		var text string
		for _, cand := range resp.Candidates {
			if cand.Content != nil {
				for _, part := range cand.Content.Parts {
					text += fmt.Sprint(part.Text)
				}
			}
		}

		return text, nil
	}

	return "", err
}

func (g *Gemini) Close() error {
	return nil
}

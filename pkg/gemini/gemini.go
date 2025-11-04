package gemini

import (
	"context"
	"fmt"

	"github.com/vgjm/linebot/pkg/llm"
	"google.golang.org/genai"
)

var _ llm.LLM = (*Gemini)(nil)

const DefaultModel = "gemini-2.5-flash"

type Gemini struct {
	ctx    context.Context
	client *genai.Client
	model  string
}

func New(ctx context.Context, apiKey string, model string) (*Gemini, error) {
	if model == "" {
		model = DefaultModel
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
		model:  model,
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

	resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(question), config)
	if err != nil {
		return "", err
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

func (g *Gemini) Close() error {
	return nil
}

package gemini

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/vgjm/linebot/pkg/llm"
	"google.golang.org/api/option"
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
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
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
	model := g.client.GenerativeModel(g.model)

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

	if instruction != "" {
		model.SystemInstruction = genai.NewUserContent(genai.Text(instruction))
	}

	resp, err := model.GenerateContent(ctx, genai.Text(question))
	if err != nil {
		return "", err
	}

	var text string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				text += fmt.Sprint(part)
			}
		}
	}
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "**", " ") // looks better

	return text, nil
}

func (g *Gemini) Close() error {
	return g.client.Close()
}

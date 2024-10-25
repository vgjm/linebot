package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/vgjm/linebot/pkg/llm"
	"google.golang.org/api/option"
)

var _ llm.LLM = (*Gemini)(nil)

const (
	PromptsEnv      = "PROMPTS"
	GeminiModel     = "GEMINI_MODEL"
	GeminiApiKeyEnv = "GEMINI_API_KEY"

	DefaultModel   = "gemini-1.5-flash"
	DefaultPrompts = `[
      {
        "text": "You are an assistant.",
        "role": "user"
      },
  	  {
        "text": "What can I do for you?",
        "role": "model"
  	  }
	]`
)

type Gemini struct {
	ctx     context.Context
	client  *genai.Client
	model   string
	history []*genai.Content
}

func New(ctx context.Context) (*Gemini, error) {
	model := os.Getenv(GeminiModel)
	if model == "" {
		model = DefaultModel
	}

	promptsStr := os.Getenv(PromptsEnv)
	if promptsStr == "" {
		promptsStr = DefaultPrompts
	}
	var prompts []prompt
	err := json.Unmarshal([]byte(promptsStr), &prompts)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompts: %w", err)
	}
	history := make([]*genai.Content, 0, 10)
	for _, p := range prompts {
		history = append(history, &genai.Content{
			Role: p.Role,
			Parts: []genai.Part{
				genai.Text(p.Text),
			},
		})
	}

	geminiApiKey := os.Getenv(GeminiApiKeyEnv)
	if geminiApiKey == "" {
		return nil, fmt.Errorf("%s is not set", GeminiApiKeyEnv)
	}
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiApiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &Gemini{
		ctx:     ctx,
		client:  client,
		model:   model,
		history: history,
	}, nil
}

func (g *Gemini) GenerateResponse(question string) (string, error) {
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

	cs := model.StartChat()
	cs.History = g.history

	resp, err := cs.SendMessage(g.ctx, genai.Text(question))
	if err != nil {
		return "Something went wrong when generating response.", err
	}

	var text string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				text += fmt.Sprint(part)
			}
		}
	}
	text = strings.ReplaceAll(text, "**", " ")
	text = strings.TrimSpace(text)
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

func (g *Gemini) Close() error {
	return g.client.Close()
}

type prompt struct {
	Role string
	Text string
}

package linebot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"google.golang.org/api/option"
)

const (
	LineChannelSecretEnv = "LINE_CHANNEL_SECRET"
	LineChannelTokenEnv  = "LINE_CHANNEL_TOKEN"
	GeminiApiKeyEnv      = "GEMINI_API_KEY"
	GeminiModel          = "GEMINI_MODEL"
	PromptsEnv           = "PROMPTS"

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

type Linebot struct {
	ctx           context.Context
	channelSecret string
	bot           *messaging_api.MessagingApiAPI
	ai            *genai.Client
	model         string
	prompts       []Prompt
}

type Prompt struct {
	Role string
	Text string
}

func New(ctx context.Context) (*Linebot, error) {
	slog.Info("Starting the linebot...")
	lineChannelSecret := os.Getenv(LineChannelSecretEnv)
	if lineChannelSecret == "" {
		return nil, fmt.Errorf("%s is not set", LineChannelSecretEnv)
	}
	lineChannelToken := os.Getenv(LineChannelTokenEnv)
	if lineChannelToken == "" {
		return nil, fmt.Errorf("%s is not set", LineChannelTokenEnv)
	}
	geminiApiKey := os.Getenv(GeminiApiKeyEnv)
	if geminiApiKey == "" {
		return nil, fmt.Errorf("%s is not set", GeminiApiKeyEnv)
	}
	model := os.Getenv(GeminiModel)
	if model == "" {
		model = DefaultModel
	}
	promptsStr := os.Getenv(PromptsEnv)
	if promptsStr == "" {
		promptsStr = DefaultPrompts
	}

	bot, err := messaging_api.NewMessagingApiAPI(lineChannelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create line bot client: %w", err)
	}

	ai, err := genai.NewClient(ctx, option.WithAPIKey(geminiApiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	var promptsJson []Prompt
	err = json.Unmarshal([]byte(promptsStr), &promptsJson)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompts: %w", err)
	}

	return &Linebot{
		ctx:           ctx,
		channelSecret: lineChannelSecret,
		bot:           bot,
		ai:            ai,
		model:         model,
		prompts:       promptsJson,
	}, nil
}

func (lb *Linebot) Close() {
	lb.ai.Close()
}

func (lb *Linebot) Callback(w http.ResponseWriter, req *http.Request) {
	cb, err := webhook.ParseRequest(lb.channelSecret, req)
	if err != nil {
		if errors.Is(err, webhook.ErrInvalidSignature) {
			slog.Warn("Received a request with invalid signature", "err", err)
			w.WriteHeader(400)
		} else {
			slog.Warn("Failed to parse the request", "err", err)
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range cb.Events {
		var err error = nil
		switch e := event.(type) {
		case webhook.MessageEvent:
			switch s := e.Source.(type) {
			case webhook.UserSource:
				err = lb.handleUserEvent(e, s)
			case webhook.GroupSource:
				err = lb.handleGroupEvent(e, s)
			default:
				err = fmt.Errorf("unkown event source: %v", e.Source.GetType())
			}
		default:
			err = fmt.Errorf("unkown event type: %v", event.GetType())
		}
		if err != nil {
			slog.Warn("Failed to handle event", "err", err, "event", event)
		}
	}

	w.WriteHeader(200)
}

func (lb *Linebot) handleUserEvent(e webhook.MessageEvent, s webhook.UserSource) error {
	slog.Info("Handling user event", "user_id", s.UserId)
	switch m := e.Message.(type) {
	case webhook.TextMessageContent:
		slog.Info("Received text message", "original_text", m.Text)
		return lb.handleTextMessage(m.Text, e.ReplyToken)
	default:
		return fmt.Errorf("unkown message type: %v", e.Message.GetType())
	}
}

func (lb *Linebot) handleGroupEvent(e webhook.MessageEvent, s webhook.GroupSource) error {
	slog.Info("Handling group event", "group_id", s.GroupId, "user_id", s.UserId)
	switch m := e.Message.(type) {
	case webhook.TextMessageContent:
		slog.Info("Received text message", "original_text", m.Text)
		if strings.HasPrefix(m.Text, "/") {
			return lb.handleTextMessage(strings.Replace(m.Text, "/", "", 1), e.ReplyToken)
		} else {
			return nil
		}
	default:
		return fmt.Errorf("unknown message type: %v", e.Message.GetType())
	}
}

func (lb *Linebot) handleTextMessage(question string, replyToken string) error {
	resp, err := lb.generateResponse(question)
	if err != nil {
		return fmt.Errorf("failed to generate response: %w", err)
	}
	if resp != "" {
		if err := lb.replyMessage(replyToken, resp); err != nil {
			return fmt.Errorf("failed to reply message: %w", err)
		}
	}
	return nil
}

func (lb *Linebot) generateResponse(question string) (string, error) {
	model := lb.ai.GenerativeModel(lb.model)

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
	contents := make([]*genai.Content, 0, 10)
	for _, prompt := range lb.prompts {
		contents = append(contents, &genai.Content{
			Role: prompt.Role,
			Parts: []genai.Part{
				genai.Text(prompt.Text),
			},
		})
	}
	cs.History = contents

	resp, err := cs.SendMessage(lb.ctx, genai.Text(question))
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

func (lb *Linebot) replyMessage(replyToken, text string) error {
	_, err := lb.bot.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text: text,
				},
			},
		},
	)

	return err
}

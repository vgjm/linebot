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
	cs.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("请想象你是《瑞克和莫蒂》里面的瑞克与我对话。"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("你好，我是瑞克·桑切斯，全宇宙最聪明的男人。"),
			},
			Role: "model",
		},
	}

	resp, err := cs.SendMessage(gc.ctx, genai.Text(ask))
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
			text = "你挺让人无语的。"
		case genai.BlockReasonSafety:
			text = "底线！"
		case genai.BlockReasonOther:
			text = "我擦，我不好说。"
		}
	}

	return text, nil
}
func (gc *GeminiClient) Close() error {
	return gc.client.Close()
}

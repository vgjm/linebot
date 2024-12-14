package linebot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/vgjm/linebot/internal/envs"
	"github.com/vgjm/linebot/internal/storage"
	"github.com/vgjm/linebot/pkg/gemini"
	"github.com/vgjm/linebot/pkg/llm"
)

type LineBot struct {
	ctx           context.Context
	channelSecret string
	messagingAPI  *messaging_api.MessagingApiAPI
	llmProvider   llm.LLM
	storage       storage.Storage
}

type LineBotConfig struct {
	Storage       storage.Storage
	ChannelSecret string
	ChannelToken  string
}

func New(ctx context.Context, cfg *LineBotConfig) (*LineBot, error) {
	slog.Info("Starting the linebot...")

	messagingAPI, err := messaging_api.NewMessagingApiAPI(cfg.ChannelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create line bot client: %w", err)
	}

	llmProvider, err := gemini.New(ctx, envs.GeminiApiKey, envs.GeminiModel)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize llm client: %w", err)
	}

	return &LineBot{
		ctx:           ctx,
		channelSecret: cfg.ChannelSecret,
		messagingAPI:  messagingAPI,
		llmProvider:   llmProvider,
		storage:       cfg.Storage,
	}, nil
}

func (lb *LineBot) Close() error {
	return lb.llmProvider.Close()
}

func (lb *LineBot) Callback(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithTimeout(req.Context(), time.Minute) // Max to 1min
	defer cancel()

	cb, err := webhook.ParseRequest(lb.channelSecret, req)
	if err != nil {
		if errors.Is(err, webhook.ErrInvalidSignature) {
			slog.Warn("Received a request with invalid signature", "error", err)
			w.WriteHeader(400)
		} else {
			slog.Warn("Failed to parse the request", "error", err)
			w.WriteHeader(500)
		}
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(cb.Events))
	for _, event := range cb.Events {
		go func() {
			defer wg.Done()
			switch e := event.(type) {
			case webhook.MessageEvent:
				switch s := e.Source.(type) {
				case webhook.UserSource:
					lb.handleUserEvent(ctx, e, s)
				case webhook.GroupSource:
					lb.handleGroupEvent(ctx, e, s)
				default:
					slog.Error("Unknown event source", "event_source", e.Source.GetType())
				}
			default:
				slog.Error("Unknown event type", "event_type", event.GetType())
			}
		}()
	}

	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	deadline, _ := ctx.Deadline()
	deadline = deadline.Add(-100 * time.Millisecond) // Exit 100ms before deadline
	timeoutChannel := time.After(time.Until(deadline))
	select {
	case <-c:
		slog.Info("Request is handled properly")
	case <-timeoutChannel:
		slog.Error("Request timeout")
	}

	w.WriteHeader(200)
}

func (lb *LineBot) handleUserEvent(ctx context.Context, e webhook.MessageEvent, s webhook.UserSource) {
	slog.Info("Handling user event", "user_id", s.UserId)
	switch m := e.Message.(type) {
	case webhook.TextMessageContent:
		slog.Info("Received text message", "original_text", m.Text)
		setting, err := lb.storage.GetUserSetting(ctx, s.UserId)
		if err != nil {
			slog.Error("Failed to get user setting", "user_id", s.UserId)
		}
		lb.handleTextMessage(ctx, setting.SystemInstruction, m.Text, e.ReplyToken, m.QuoteToken)
	default:
		slog.Error("Unknown message type", "message_type", e.Message.GetType())
	}
}

func (lb *LineBot) handleGroupEvent(ctx context.Context, e webhook.MessageEvent, s webhook.GroupSource) {
	slog.Info("Handling group event", "group_id", s.GroupId, "user_id", s.UserId)
	switch m := e.Message.(type) {
	case webhook.TextMessageContent:
		slog.Info("Received text message", "original_text", m.Text)
		if strings.HasPrefix(m.Text, "/") {
			setting, err := lb.storage.GetGroupUserSetting(ctx, s.GroupId, s.UserId)
			if err != nil {
				slog.Error("Failed to get group user setting", "group_id", s.GroupId, "user_id", s.UserId)
			}
			lb.handleTextMessage(ctx, setting.SystemInstruction, strings.Replace(m.Text, "/", "", 1), e.ReplyToken, m.QuoteToken)
		}
	default:
		slog.Error("Unknown message type", "message_type", e.Message.GetType())
	}
}

func (lb *LineBot) handleTextMessage(ctx context.Context, instruction, question,
	replyToken, quoteToken string) {

	respChannel := make(chan string)
	go func() {
		resp, err := lb.llmProvider.GenerateContent(ctx, instruction, question)
		if err != nil {
			slog.Error("Failed to generate response", "error", err)
			resp = "Something went wrong when generating response"
		}
		respChannel <- resp
	}()

	deadline, _ := ctx.Deadline()
	deadline = deadline.Add(-1 * time.Second) // leave some time to inform users
	timeoutChannel := time.After(time.Until(deadline))

	select {
	case resp := <-respChannel:
		if resp != "" {
			if err := lb.replyMessage(resp, replyToken, quoteToken); err != nil {
				slog.Error("Failed to reply message", "error", err)
			}
		}
	case <-timeoutChannel:
		if err := lb.replyMessage("Timeout when generating response", replyToken, quoteToken); err != nil {
			slog.Error("Failed to reply message", "error", err)
		}
	}
}

func (lb *LineBot) replyMessage(text, replyToken, quoteToken string) error {
	_, err := lb.messagingAPI.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text:       text,
					QuoteToken: quoteToken,
				},
			},
		},
	)

	return err
}

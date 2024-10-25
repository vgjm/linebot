package linebot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/vgjm/linebot/pkg/gemini"
	"github.com/vgjm/linebot/pkg/llm"
)

const (
	LineChannelSecretEnv = "LINE_CHANNEL_SECRET"
	LineChannelTokenEnv  = "LINE_CHANNEL_TOKEN"
)

type Linebot struct {
	ctx           context.Context
	channelSecret string
	bot           *messaging_api.MessagingApiAPI
	ai            llm.LLM
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

	bot, err := messaging_api.NewMessagingApiAPI(lineChannelToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create line bot client: %w", err)
	}

	ai, err := gemini.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize llm client: %w", err)
	}

	return &Linebot{
		ctx:           ctx,
		channelSecret: lineChannelSecret,
		bot:           bot,
		ai:            ai,
	}, nil
}

func (lb *Linebot) Close() error {
	return lb.ai.Close()
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
			slog.Warn("Failed to handle event", "err", err)
		}
	}

	w.WriteHeader(200)
}

func (lb *Linebot) handleUserEvent(e webhook.MessageEvent, s webhook.UserSource) error {
	slog.Info("Handling user event", "user_id", s.UserId)
	switch m := e.Message.(type) {
	case webhook.TextMessageContent:
		slog.Info("Received text message", "original_text", m.Text)
		return lb.handleTextMessage(m.Text, e.ReplyToken, m.QuoteToken)
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
			return lb.handleTextMessage(strings.Replace(m.Text, "/", "", 1), e.ReplyToken, m.QuoteToken)
		} else {
			return nil
		}
	default:
		return fmt.Errorf("unknown message type: %v", e.Message.GetType())
	}
}

func (lb *Linebot) handleTextMessage(question string, replyToken string, quoteToken string) error {
	resp, err := lb.ai.GenerateResponse(question)
	if resp != "" {
		if err := lb.replyMessage(resp, replyToken, quoteToken); err != nil {
			return fmt.Errorf("failed to reply message: %w", err)
		}
	}
	if err != nil {

		return fmt.Errorf("failed to generate response: %w", err)
	}

	return nil
}

func (lb *Linebot) replyMessage(text, replyToken, quoteToken string) error {
	_, err := lb.bot.ReplyMessage(
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

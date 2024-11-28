package linebot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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
		w.WriteHeader(200)
	case <-timeoutChannel:
		w.WriteHeader(408) //Timeout
	}
}

func (lb *Linebot) handleUserEvent(ctx context.Context, e webhook.MessageEvent, s webhook.UserSource) {
	slog.Info("Handling user event", "user_id", s.UserId)
	switch m := e.Message.(type) {
	case webhook.TextMessageContent:
		slog.Info("Received text message", "original_text", m.Text)
		lb.handleTextMessage(ctx, m.Text, e.ReplyToken, m.QuoteToken)
	default:
		slog.Error("Unknown message type", "message_type", e.Message.GetType())
	}
}

func (lb *Linebot) handleGroupEvent(ctx context.Context, e webhook.MessageEvent, s webhook.GroupSource) {
	slog.Info("Handling group event", "group_id", s.GroupId, "user_id", s.UserId)
	switch m := e.Message.(type) {
	case webhook.TextMessageContent:
		slog.Info("Received text message", "original_text", m.Text)
		if strings.HasPrefix(m.Text, "/") {
			lb.handleTextMessage(ctx, strings.Replace(m.Text, "/", "", 1), e.ReplyToken, m.QuoteToken)
		}
	default:
		slog.Error("Unknown message type", "message_type", e.Message.GetType())
	}
}

func (lb *Linebot) handleTextMessage(ctx context.Context, question string, replyToken string, quoteToken string) {

	respChannel := make(chan string)
	go func() {
		resp, err := lb.ai.GenerateResponse(ctx, question)
		if err != nil {
			slog.Error("Failed to generate response", "error", err)
			resp = "Something went wrong when generating response"
		}
		respChannel <- resp
	}()

	deadline, _ := ctx.Deadline()
	deadline = deadline.Add(-500 * time.Millisecond)
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

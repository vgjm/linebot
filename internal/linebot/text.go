package linebot

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/vgjm/linebot/internal/storage"
)

type MessageSource int

const (
	UserSource MessageSource = iota
	GroupSource
)

type TextMessageMeta struct {
	Type       MessageSource
	UserId     string
	GroupId    string
	Text       string
	ReplyToken string
	QuoteToken string
}

func (lb *LineBot) GetInstruct(ctx context.Context, meta TextMessageMeta, groupDefault bool) (string, error) {
	var instruct string
	switch meta.Type {
	case UserSource:
		setting, err := lb.storage.GetUserSetting(ctx, meta.UserId)
		if err != nil {
			return "", err
		}
		instruct = setting.SystemInstruction
	case GroupSource:
		setting, err := lb.storage.GetGroupUserSetting(ctx, meta.GroupId, meta.UserId)
		if err != nil {
			return "", err
		}
		if groupDefault && setting.SystemInstruction == "" {
			setting, err = lb.storage.GetGroupUserSetting(ctx, meta.GroupId, "default")
			if err != nil {
				return "", err
			}
		}
		instruct = setting.SystemInstruction
	}
	return instruct, nil
}

func (lb *LineBot) SetInstruct(ctx context.Context, meta TextMessageMeta, instruct string) error {
	var err error
	switch meta.Type {
	case UserSource:
		err = lb.storage.UpsertUserSetting(ctx, storage.UserSetting{
			UserId:            meta.UserId,
			SystemInstruction: instruct,
		})
	case GroupSource:
		err = lb.storage.UpsertGroupUserSetting(ctx, storage.GroupUserSetting{
			GroupId:           meta.GroupId,
			UserId:            meta.UserId,
			SystemInstruction: instruct,
		})
	}
	return err
}

func (lb *LineBot) handleTextMessage(ctx context.Context, meta TextMessageMeta) {
	if !lb.handleInstruction(ctx, meta) {
		lb.generateContent(ctx, meta)
	}
}

func (lb *LineBot) handleInstruction(ctx context.Context, meta TextMessageMeta) bool {
	if strings.HasPrefix(meta.Text, "set") ||
		strings.HasPrefix(meta.Text, "get") {
		tokens := strings.Split(meta.Text, " ")
		if len(tokens) >= 2 {
			switch tokens[0] {
			case "set":
				switch tokens[1] {
				case "instruction":
					instruct := strings.Replace(meta.Text, "set instruction ", "", 1)
					if err := lb.SetInstruct(ctx, meta, instruct); err != nil {
						slog.Error("Failed to set instruction", "user_id", meta.UserId, "group_id", meta.GroupId, "instruction", instruct)
					} else {
						if err := lb.replyMessage("instruction updated", meta.ReplyToken, meta.QuoteToken); err != nil {
							slog.Error("Failed to reply message", "error", err)
						}
					}
					return true
				}
			case "get":
				switch tokens[1] {
				case "instruction":
					reply, err := lb.GetInstruct(ctx, meta, false)
					if err != nil {
						slog.Error("Failed to get instruction", "user_id", meta.UserId, "group_id", meta.GroupId)
						reply = "Something went wrong when fetching your instruction"
					}
					if err := lb.replyMessage(reply, meta.ReplyToken, meta.QuoteToken); err != nil {
						slog.Error("Failed to reply message", "error", err)
					}
					return true
				}
				return true
			}
		}
	}
	return false
}

func (lb *LineBot) generateContent(ctx context.Context, meta TextMessageMeta) {
	instruct, err := lb.GetInstruct(ctx, meta, true)
	if err != nil {
		slog.Error("Failed to get instruction", "user_id", meta.UserId, "group_id", meta.GroupId)
	}

	respChannel := make(chan string)
	go func() {
		resp, err := lb.llmProvider.GenerateContent(ctx, instruct, meta.Text)
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
			if err := lb.replyMessage(resp, meta.ReplyToken, meta.QuoteToken); err != nil {
				slog.Error("Failed to reply message", "error", err)
			}
		}
	case <-timeoutChannel:
		if err := lb.replyMessage("Timeout when generating response", meta.ReplyToken, meta.QuoteToken); err != nil {
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

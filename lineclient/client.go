package lineclient

import (
	"log"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
)

type LineClient struct {
	api *messaging_api.MessagingApiAPI
}

func New(token string) (*LineClient, error) {
	api, err := messaging_api.NewMessagingApiAPI(token)
	if err != nil {
		return nil, err
	}
	return &LineClient{
		api: api,
	}, nil
}

func (client *LineClient) ReplyMessage(replyToken string, text string) error {
	if _, err := client.api.ReplyMessage(
		&messaging_api.ReplyMessageRequest{
			ReplyToken: replyToken,
			Messages: []messaging_api.MessageInterface{
				messaging_api.TextMessage{
					Text: text,
				},
			},
		},
	); err != nil {
		return err
	} else {
		log.Println("Sent text reply.")
	}
	return nil
}

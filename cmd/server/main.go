package main

import (
	"context"
	"log"
	"net/http"

	"github.com/vgjm/linebot/internal/dynamodriver"
	"github.com/vgjm/linebot/internal/envs"
	"github.com/vgjm/linebot/internal/linebot"
)

func main() {
	ctx := context.Background()

	storageDriver, err := dynamodriver.New(ctx, dynamodriver.Config{})
	if err != nil {
		log.Fatalf("Failed to create dynamodb client: %v\n", err)
	}
	lb, err := linebot.New(ctx, &linebot.LineBotConfig{
		Storage:       storageDriver,
		ChannelSecret: envs.LineChannelSecret,
		ChannelToken:  envs.LineChannelToken,
	})
	if err != nil {
		log.Fatalf("Failed to create line bot client: %v\n", err)
	}
	defer lb.Close()

	http.HandleFunc("/", lb.Callback)

	http.ListenAndServe(":5000", nil)

}

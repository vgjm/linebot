package main

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
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
		log.Fatal(err)
	}
	defer lb.Close()

	http.HandleFunc("/", lb.Callback)

	lambda.Start(httpadapter.New(http.DefaultServeMux).ProxyWithContext)
}

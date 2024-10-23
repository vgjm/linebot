package main

import (
	"context"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/vgjm/linebot/linebot"
)

func main() {
	ctx := context.Background()

	lb, err := linebot.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer lb.Close()

	http.HandleFunc("/", lb.Callback)

	lambda.Start(httpadapter.New(http.DefaultServeMux).ProxyWithContext)
}

package main

import (
	"context"
	"log"
	"net/http"

	"github.com/vgjm/linebot/internal/linebot"
)

func main() {
	ctx := context.Background()

	lb, err := linebot.New(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer lb.Close()

	http.HandleFunc("/", lb.Callback)

	http.ListenAndServe(":5000", nil)

}

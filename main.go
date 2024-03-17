// Copyright 2016 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vgjm/linebot/geminiclient"
	"github.com/vgjm/linebot/lineclient"
	"github.com/vgjm/linebot/router"
)

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	channelSecret := os.Getenv("LINE_CHANNEL_SECRET")

	lineClient, err := lineclient.New(os.Getenv("LINE_CHANNEL_TOKEN"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start Line client")
	}

	geminiClient, err := geminiclient.New(os.Getenv("GEMINI_API_KEY"))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start Gemini client")
	}
	defer geminiClient.Close()

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		log.Debug().Msg("/callback called...")

		cb, err := webhook.ParseRequest(channelSecret, req)
		if err != nil {
			log.Err(err).Msg("Cannot parse request")
			if errors.Is(err, webhook.ErrInvalidSignature) {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}

		log.Debug().Msg("Handling events...")
		for _, event := range cb.Events {
			log.Debug().Msgf("Event type: %v", event.GetType())

			switch e := event.(type) {
			case webhook.MessageEvent:
				switch message := e.Message.(type) {
				case webhook.TextMessageContent:
					mType := router.Route(message.Text)
					switch mType {
					case router.MENU:
						if err := lineClient.ReplyMessage(e.ReplyToken, "菜单"); err != nil {
							log.Err(err).Msg("Failed to reply message")
						}
					case router.AI_REPLY:
						resp, err := geminiClient.SingleQuestion(strings.Replace(message.Text, "/", "", 1))
						if err != nil {
							log.Err(err).Msg("Failed to call Gemini API")
							resp = "Genmini已麻。"
						}
						if err := lineClient.ReplyMessage(e.ReplyToken, resp); err != nil {
							log.Err(err).Msg("Failed to reply message")
						}
					}
				default:
					log.Info().Msgf("Unsupported message content: %T\n", e.Message)
				}
			default:
				log.Info().Msgf("Unsupported message: %T\n", event)
			}
		}
	})

	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	fmt.Println("http://localhost:" + port + "/")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal().Err(err)
	}
}

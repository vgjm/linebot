package core

import (
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/vgjm/linebot/geminiclient"
	"github.com/vgjm/linebot/lineclient"
)

const (
	lineSecret = "LINE_CHANNEL_SECRET"
	lineToken  = "LINE_CHANNEL_TOKEN"
	geminiKey  = "GEMINI_API_KEY"
	prompts    = "PROMPTS"
)

var lineClient *lineclient.LineClient
var geminiClient *geminiclient.GeminiClient

func Start() {
	/**
	 * Init logger
	 */
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	/**
	 * Init line client and gemini client
	 */
	var err error
	lineClient, err = lineclient.New(os.Getenv(lineToken))
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start Line client")
	}
	geminiClient, err = geminiclient.New(os.Getenv(geminiKey), prompts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start Gemini client")
	}

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/", lineEventHandler)

	// http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
	// 	log.Info().Msg("healthcheck called...")
	// 	w.Header().Set("Content-Type", "application/json")
	// 	fmt.Fprintf(w, `{ "status": "OK" }`)
	// })

	// Start lambda
	lambda.Start(httpadapter.New(http.DefaultServeMux).ProxyWithContext)

	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	// port := os.Getenv("PORT")
	// if port == "" {
	// 	port = "5000"
	// }
	// log.Info().Msg("http://localhost:" + port + "/")
	// if err := http.ListenAndServe(":"+port, nil); err != nil {
	// 	log.Fatal().Err(err)
	// }
}

func lineEventHandler(w http.ResponseWriter, req *http.Request) {
	log.Info().Msg("/line handler called...")

	channelSecret := os.Getenv(lineSecret)
	cb, err := webhook.ParseRequest(channelSecret, req)
	if err != nil {
		log.Err(err).Msg("Cannot parse request")
		if errors.Is(err, webhook.ErrInvalidSignature) {
			w.WriteHeader(403)
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
				switch e.Source.GetType() {
				case "user":
					log.Info().Msg("Generating response message for user chat...")
					resp, err := geminiClient.Generate(message.Text)
					if err != nil {
						log.Err(err).Msg("Failed to call Gemini API")
					}
					log.Info().Msgf("Response message generated: %s", resp)
					if resp != "" {
						if err := lineClient.ReplyMessage(e.ReplyToken, resp); err != nil {
							log.Err(err).Msg("Failed to reply message")
						}
					}
				case "group":
					if !strings.HasPrefix(message.Text, "/") {
						break
					}
					log.Info().Msg("Generating response message for group chat...")
					resp, err := geminiClient.Generate(strings.Replace(message.Text, "/", "", 1))
					if err != nil {
						log.Err(err).Msg("Failed to call Gemini API")
					}
					log.Info().Msgf("Response message generated: %s", resp)
					if resp != "" {
						if err := lineClient.ReplyMessage(e.ReplyToken, resp); err != nil {
							log.Err(err).Msg("Failed to reply message")
						}
					}
				default:
					log.Debug().Msgf("Unknown event source: %v", e.Source.GetType())
				}
			default:
				log.Debug().Msgf("Unknown message type: %v", e.Message)
			}
		default:
			log.Debug().Msgf("Unknown event type: %v", event)
		}
	}

	io.WriteString(w, "OK")
}

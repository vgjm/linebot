package envs

import (
	"os"
)

const (
	lineChannelSecretEnv = "LINE_CHANNEL_SECRET"
	lineChannelTokenEnv  = "LINE_CHANNEL_TOKEN"
	geminiModelEnv       = "GEMINI_MODEL"
	geminiApiKeyEnv      = "GEMINI_API_KEY"

)

var (
	LineChannelSecret string
	LineChannelToken  string
	GeminiApiKey      string
	GeminiModel       string
)

func init() {
	LineChannelSecret = os.Getenv(lineChannelSecretEnv)
	LineChannelToken = os.Getenv(lineChannelTokenEnv)
	GeminiModel = os.Getenv(geminiModelEnv)
	GeminiApiKey = os.Getenv(geminiApiKeyEnv)
}

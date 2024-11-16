package llm

import "context"

type LLM interface {
	GenerateResponse(context.Context, string) (string, error)
	Close() error
}

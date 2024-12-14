package llm

import "context"

type LLM interface {
	GenerateContent(ctx context.Context, instruction string, question string) (string, error)
	Close() error
}

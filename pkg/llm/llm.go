package llm

type LLM interface {
	GenerateResponse(string) (string, error)
	Close() error
}

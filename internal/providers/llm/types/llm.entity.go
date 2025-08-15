package providers

type LlmResponseEntity struct {
	Message     string
	Model       string
	TokensUsed  int
	ContextSize int
}

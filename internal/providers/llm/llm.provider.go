package providers

import (
	llm_type "github.com/PedroHercules/gommit/internal/providers/llm/types"
	"github.com/PedroHercules/gommit/internal/types"
)

type LlmProvider interface {
	GenerateCommitMessage(diff string) *types.ResultEntity[llm_type.LlmResponseEntity]
	GeneratePrMessage(diff string) *types.ResultEntity[llm_type.LlmResponseEntity]
}

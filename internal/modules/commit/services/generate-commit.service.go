package commit

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	llm_provider "github.com/PedroHercules/gommit/internal/providers/llm"
	llm_type "github.com/PedroHercules/gommit/internal/providers/llm/types"
	"github.com/PedroHercules/gommit/internal/types"
)

func runGitDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--staged")

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	diff := string(output)
	if strings.TrimSpace(diff) == "" {
		return "", fmt.Errorf("nenhuma alteração encontrada no stage. Execute 'git add <arquivo>' para adicionar arquivos ao stage antes de gerar o commit")
	}

	return diff, nil
}

func askForConfirmation(message string, model string, tokensUsed int, contextSize int) bool {
	fmt.Println("\n=== Informações do LLM ===")
	fmt.Printf("Modelo: %s\n", model)
	fmt.Printf("Tokens utilizados: %d\n", tokensUsed)
	fmt.Printf("Tamanho do contexto: %d\n", contextSize)
	fmt.Println("=========================")
	
	fmt.Println("\n=== Mensagem de Commit Gerada ===")
	fmt.Println(message)
	fmt.Println("=================================")
	fmt.Print("\nDeseja prosseguir com este commit? (s/N): ")
	
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))
	
	return response == "s" || response == "sim" || response == "y" || response == "yes"
}

func runGitCommit(message string, model string, tokensUsed int, contextSize int) error {
	if !askForConfirmation(message, model, tokensUsed, contextSize) {
		fmt.Println("\nCommit cancelado pelo usuário.")
		return fmt.Errorf("commit cancelado pelo usuário")
	}
	
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.Output()
	if err != nil {
		return err
	}
	fmt.Println("\n=== Commit Realizado ===")
	fmt.Println(string(output))
	return nil
}

func GenerateCommit(llmProvider *llm_provider.OpenRouterProvider) *types.ResultEntity[llm_type.LlmResponseEntity] {

	diff, err := runGitDiff()
	if err != nil {
		return types.NewError[llm_type.LlmResponseEntity](err)
	}

	fmt.Println("🤖 Generating commit message with AI...")
	generateCommitResponse := llmProvider.GenerateCommitMessage(diff)
	if generateCommitResponse.IsFailure() {
		return generateCommitResponse
	}

	responseData := generateCommitResponse.GetData()
	commitMessage := responseData.Message
	err = runGitCommit(commitMessage, responseData.Model, responseData.TokensUsed, responseData.ContextSize)
	if err != nil {
		return types.NewError[llm_type.LlmResponseEntity](err)
	}

	return types.NewSuccess(llm_type.LlmResponseEntity{
		Message: commitMessage,
	})
}

package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/repositories"
)

// GrokCLIRepository delegates OAuth-backed generation to the official Grok CLI.
type GrokCLIRepository struct {
	binary  string
	timeout time.Duration
}

func NewGrokCLIRepository() *GrokCLIRepository {
	return &GrokCLIRepository{binary: "grok", timeout: 3 * time.Minute}
}

func (r *GrokCLIRepository) ConfigureProvider(provider, authMethod string) error {
	if provider != "grok" || authMethod != "oauth" {
		return fmt.Errorf("Grok CLI supports only grok/oauth, got %s/%s", provider, authMethod)
	}
	if _, err := exec.LookPath(r.binary); err != nil {
		return errors.New("Grok CLI was not found; install the official `@xai-official/grok` package and run `grok login`")
	}
	return nil
}

func (r *GrokCLIRepository) GenerateCommitMessage(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	return r.generate(model, (&OpenRouterRepository{}).createCommitPrompt(diff), (&OpenRouterRepository{}).cleanCommitMessage)
}

func (r *GrokCLIRepository) GeneratePRDescription(diff *entities.GitDiff, model string) (*repositories.LLMResponse, error) {
	return r.generate(model, (&OpenRouterRepository{}).createPRPrompt(diff), (&OpenRouterRepository{}).cleanPRDescription)
}

func (r *GrokCLIRepository) generate(model, prompt string, clean func(string) string) (*repositories.LLMResponse, error) {
	if strings.TrimSpace(model) == "" {
		return failedLLMResponse(errors.New("Grok model is not configured; run `gmit config` to select one")), nil
	}
	if _, err := exec.LookPath(r.binary); err != nil {
		return failedLLMResponse(errors.New("Grok CLI is missing; install the official CLI and run `grok login`")), nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()
	args := []string{
		"--no-auto-update", "-p", prompt, "--model", model,
		"--output-format", "json", "--tools", "", "--no-subagents",
		"--disable-web-search", "--max-turns", "1",
	}
	cmd := exec.CommandContext(ctx, r.binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return failedLLMResponse(fmt.Errorf("Grok CLI timed out after %s", r.timeout)), nil
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return failedLLMResponse(fmt.Errorf("Grok CLI failed: %s", truncateErrorBody([]byte(message), 512))), nil
	}
	text, err := extractGrokCLIText(stdout.Bytes())
	if err != nil {
		return failedLLMResponse(err), nil
	}
	message := clean(text)
	if message == "" {
		return failedLLMResponse(errors.New("Grok CLI returned an empty response")), nil
	}
	return &repositories.LLMResponse{Message: message, Model: model, Success: true}, nil
}

func extractGrokCLIText(output []byte) (string, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return "", errors.New("Grok CLI returned no output")
	}
	var decoded any
	if json.Unmarshal(trimmed, &decoded) == nil {
		if result := findGrokText(decoded); result != "" {
			return result, nil
		}
		return "", errors.New("Grok CLI JSON output contained no assistant text")
	}
	return string(trimmed), nil
}

func findGrokText(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		for _, key := range []string{"output_text", "result", "response", "text", "content", "message"} {
			if nested, ok := typed[key]; ok {
				if text := findGrokText(nested); text != "" {
					return text
				}
			}
		}
		for key, nested := range typed {
			if key == "metadata" || key == "usage" || key == "model" || key == "id" {
				continue
			}
			if text := findGrokText(nested); text != "" {
				return text
			}
		}
	case []any:
		var parts []string
		for _, item := range typed {
			if text := findGrokText(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case string:
		return typed
	}
	return ""
}

var grokModelIDPattern = regexp.MustCompile(`\bgrok-[A-Za-z0-9][A-Za-z0-9._-]*\b`)

func (r *GrokCLIRepository) GetAvailableModels() ([]repositories.LLMModel, error) {
	if _, err := exec.LookPath(r.binary); err != nil {
		return nil, errors.New("Grok CLI was not found; install the official CLI to use OAuth")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.binary, "models")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("could not list Grok models; run `grok login`: %s", truncateErrorBody([]byte(message), 512))
	}
	models := parseGrokModels(stdout.Bytes())
	if len(models) == 0 {
		return nil, errors.New("Grok CLI returned no recognizable model IDs")
	}
	return models, nil
}

func parseGrokModels(output []byte) []repositories.LLMModel {
	seen := make(map[string]bool)
	models := make([]repositories.LLMModel, 0)
	for _, match := range grokModelIDPattern.FindAllString(string(output), -1) {
		if seen[match] {
			continue
		}
		seen[match] = true
		models = append(models, repositories.LLMModel{ID: match, Name: match, Provider: "Grok", Available: true})
	}
	return models
}

func (r *GrokCLIRepository) ValidateModel(modelID string) (bool, error) {
	models, err := r.GetAvailableModels()
	if err != nil {
		return false, err
	}
	for _, model := range models {
		if model.ID == modelID {
			return true, nil
		}
	}
	return false, nil
}

func (r *GrokCLIRepository) GetBestModel() (*repositories.LLMModel, error) {
	return nil, errors.New("select a Grok model with `gmit config`")
}

func (r *GrokCLIRepository) TestConnection(string) (bool, error) {
	_, err := r.GetAvailableModels()
	return err == nil, err
}

func (r *GrokCLIRepository) SetAPIKey(string) {}

func (r *GrokCLIRepository) GetModelInfo(modelID string) (*repositories.LLMModel, error) {
	models, err := r.GetAvailableModels()
	if err != nil {
		return nil, err
	}
	for _, model := range models {
		if model.ID == modelID {
			return &model, nil
		}
	}
	return nil, fmt.Errorf("Grok model not found: %s", modelID)
}

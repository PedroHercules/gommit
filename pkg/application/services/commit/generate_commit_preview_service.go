package services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

type generateCommitPreviewService struct {
	generateCommitUseCase *usecases.GenerateCommitUseCase
}

type GenerateCommitPreviewDTO struct {
	Model  string
	DryRun bool
	Force  bool
}

type GenerateCommitPreviewEntity struct {
	CommitMessage  string
	Model          string
	TokensUsed     int
	Warnings       []string
	Success        bool
	ErrorMessage   string
	ChangesSummary string
}

func newGenerateCommitPreviewService(generateCommitUseCase *usecases.GenerateCommitUseCase) *generateCommitPreviewService {
	return &generateCommitPreviewService{
		generateCommitUseCase: generateCommitUseCase,
	}
}

func (commitService *generateCommitPreviewService) GenerateCommitPreview(data GenerateCommitPreviewDTO) (*GenerateCommitPreviewEntity, error) {
	response := &GenerateCommitPreviewEntity{
		Warnings: []string{},
	}

	generateRequest := usecases.GenerateCommitRequest{
		Model: data.Model,
		Force: data.Force,
	}

	generateResponse, generateCommitErr := commitService.generateCommitUseCase.Execute(generateRequest)
	if generateCommitErr != nil {
		response.ErrorMessage = generateCommitErr.Error()
		return response, nil
	}

	if !generateResponse.Success {
		response.ErrorMessage = generateResponse.ErrorMessage
		return response, nil
	}

	response.CommitMessage = generateResponse.Commit.Message
	response.Model = generateResponse.LLMResponse.Model
	response.TokensUsed = generateResponse.LLMResponse.TokensUsed
	response.Warnings = generateResponse.Warnings
	response.ChangesSummary = generateResponse.Diff.GetChangesSummary()
	response.Success = true

	return response, nil
}

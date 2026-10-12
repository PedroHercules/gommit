package commit_services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

type ValidateCommitMessageResult struct {
	Valid          bool
	IsConventional bool
	Warnings       []string
	Suggestions    []string
	ErrorMessage   string
}

type ServiceContainer struct {
	generateCommitPreviewService *generateCommitPreviewService
	confirmCommitService         *ConfirmCommitService
}

type CommitService struct {
	services *ServiceContainer
}

func NewCommitService(
	generateCommitUseCase *usecases.GenerateCommitUseCase,
	commitUseCase *usecases.CommitUseCase,
) *CommitService {
	container := &ServiceContainer{
		generateCommitPreviewService: newGenerateCommitPreviewService(generateCommitUseCase),
		confirmCommitService:         newConfirmCommitService(commitUseCase),
	}

	return &CommitService{
		services: container,
	}
}

func (c *CommitService) GenerateCommitPreview(data GenerateCommitPreviewDTO) (*GenerateCommitPreviewEntity, error) {
	return c.services.generateCommitPreviewService.GenerateCommitPreview(data)
}

func (c *CommitService) ConfirmCommit(data ConfirmCommitDTO) (*ConfirmCommitEntity, error) {
	return c.services.confirmCommitService.ConfirmCommit(data)
}

func (c *CommitService) ValidateCommitMessage(message string) (*ValidateCommitMessageResult, error) {
	result, err := c.services.confirmCommitService.confirmCommitUseCase.ValidateCommitMessage(usecases.ValidateCommitMessageRequest{Message: message})
	if err != nil {
		return nil, err
	}
	return &ValidateCommitMessageResult{
		Valid:          result.Valid,
		IsConventional: result.IsConventional,
		Warnings:       result.Warnings,
		Suggestions:    result.Suggestions,
		ErrorMessage:   result.ErrorMessage,
	}, nil
}

func (c *CommitService) PushCurrentBranch() error {
	return c.services.confirmCommitService.PushCurrentBranch()
}

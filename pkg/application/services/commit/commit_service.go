package commit_services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

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

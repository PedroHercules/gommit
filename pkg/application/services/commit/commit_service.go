package services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

type ServiceContainer struct {
	generateCommitPreviewService *generateCommitPreviewService
}

type CommitService struct {
	services *ServiceContainer
}

func NewCommitService(useCases ...*usecases.GenerateCommitUseCase) *CommitService {
	container := &ServiceContainer{
		generateCommitPreviewService: newGenerateCommitPreviewService(useCases[0]),
	}

	return &CommitService{
		services: container,
	}
}

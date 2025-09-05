package pr_services

import "github.com/PedroHercules/gommit/pkg/domain/usecases"

type ServiceContainer struct {
	generatePRPreviewService *GeneratePRPreviewService
}

func NewPullRequestService(prUseCase *usecases.PullRequestUseCase) *ServiceContainer {
	return &ServiceContainer{
		generatePRPreviewService: NewGeneratePRPreviewService(prUseCase),
	}
}

func (c *ServiceContainer) GeneratePRPreview(request *GeneratePRPreviewRequest) (*GeneratePRPreviewResponse, error) {
	return c.generatePRPreviewService.GeneratePreview(request)
}

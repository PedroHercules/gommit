package pr_services

import (
	"fmt"

	"github.com/PedroHercules/gommit/pkg/domain/entities"
	"github.com/PedroHercules/gommit/pkg/domain/usecases"
)

type GeneratePRPreviewService struct {
	prUseCase *usecases.PullRequestUseCase
}

func NewGeneratePRPreviewService(prUseCase *usecases.PullRequestUseCase) *GeneratePRPreviewService {
	return &GeneratePRPreviewService{
		prUseCase: prUseCase,
	}
}

type GeneratePRPreviewRequest struct {
	BaseBranch string
}

type GeneratePRPreviewResponse struct {
	Success      bool
	Message      string
	ErrorMessage string
	PullRequest  *entities.PullRequest
}

func (s *GeneratePRPreviewService) GeneratePreview(request *GeneratePRPreviewRequest) (*GeneratePRPreviewResponse, error) {
	response := &GeneratePRPreviewResponse{}

	// Step 1: Validate base branch
	if request.BaseBranch == "" {
		response.ErrorMessage = "base branch cannot be empty"
		return response, nil
	}

	// Step 2: Generate preview
	req := &usecases.GeneratePullRequestRequest{
		BaseBranch: request.BaseBranch,
	}

	pr, err := s.prUseCase.GeneratePRPreview(req)
	if err != nil {
		response.ErrorMessage = fmt.Sprintf("Failed to generate preview: %v", err)
		return response, nil
	}

	// Step 3: Set response
	response.Success = true
	response.Message = "Preview generated successfully"
	response.PullRequest = pr.PullRequest

	return response, nil
}

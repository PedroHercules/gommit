package services

import (
	"github.com/PedroHercules/gommit/pkg/domain/usecases"
)

type ConfirmCommitDTO struct {
	Message string
	DryRun  bool
}

type ConfirmCommitEntity struct {
	Success      bool
	ErrorMessage string
}

type ConfirmCommitService struct {
	confirmCommitUseCase *usecases.CommitUseCase
}

func newConfirmCommitService(commitUseCase *usecases.CommitUseCase) *ConfirmCommitService {
	return &ConfirmCommitService{
		confirmCommitUseCase: commitUseCase,
	}
}

func (commitService *ConfirmCommitService) ConfirmCommit(data ConfirmCommitDTO) (*ConfirmCommitEntity, error) {
	response := &ConfirmCommitEntity{}

	commitRequest := usecases.CommitChangesRequest{
		Message: data.Message,
		DryRun:  data.DryRun,
	}

	commitResponse, err := commitService.confirmCommitUseCase.CommitChanges(commitRequest)
	if err != nil {
		response.ErrorMessage = err.Error()
		return response, nil
	}

	if !commitResponse.Success {
		response.ErrorMessage = commitResponse.ErrorMessage
		return response, nil
	}

	response.Success = true
	return response, nil
}

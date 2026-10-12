package usecases

import "testing"

func TestValidateCommitMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		valid        bool
		conventional bool
		wantWarnings bool
	}{
		{name: "conventional message", message: "feat(auth): add token refresh", valid: true, conventional: true},
		{name: "non conventional message", message: "update the login page", valid: true, conventional: false, wantWarnings: true},
		{name: "empty message", message: "", valid: false},
	}
	useCase := NewCommitUseCase(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := useCase.ValidateCommitMessage(ValidateCommitMessageRequest{Message: tt.message})
			if err != nil {
				t.Fatalf("ValidateCommitMessage() error = %v", err)
			}
			if response.Valid != tt.valid || response.IsConventional != tt.conventional {
				t.Fatalf("got valid=%v conventional=%v, want valid=%v conventional=%v", response.Valid, response.IsConventional, tt.valid, tt.conventional)
			}
			if gotWarnings := len(response.Warnings) > 0; gotWarnings != tt.wantWarnings {
				t.Fatalf("got warnings=%v, want warnings=%v (%v)", gotWarnings, tt.wantWarnings, response.Warnings)
			}
		})
	}
}

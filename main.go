package main

import (
	"fmt"
	"os"

	commit_services "github.com/PedroHercules/gommit/pkg/application/services/commit"
	config_services "github.com/PedroHercules/gommit/pkg/application/services/config"
	pr_services "github.com/PedroHercules/gommit/pkg/application/services/pull-request"
	"github.com/PedroHercules/gommit/pkg/domain/usecases"
	"github.com/PedroHercules/gommit/pkg/infrastructure/config"
	"github.com/PedroHercules/gommit/pkg/infrastructure/git"
	"github.com/PedroHercules/gommit/pkg/infrastructure/llm"
	"github.com/PedroHercules/gommit/pkg/interfaces/cli"
)

func main() {
	configRepo, err := config.NewFileConfigRepository()
	if err != nil {
		fmt.Printf("Error creating config repository: %v\n", err)
		os.Exit(1)
	}

	gitRepo, err := git.NewCommandGitRepository("")
	if err != nil {
		fmt.Printf("Error creating git repository: %v\n", err)
		os.Exit(1)
	}

	llmRepo := llm.NewProviderRouter(
		llm.NewOpenRouterRepository(""),
		llm.NewGrokAPIRepository(""),
		llm.NewGrokCLIRepository(),
	)

	configUseCase := usecases.NewConfigUseCase(configRepo, llmRepo)
	generateCommitUseCase := usecases.NewGenerateCommitUseCase(gitRepo, llmRepo, configRepo)
	commitUseCase := usecases.NewCommitUseCase(gitRepo)
	pullRequestUseCase := usecases.NewPullRequestUseCase(gitRepo, llmRepo, configRepo)

	commitService := commit_services.NewCommitService(generateCommitUseCase, commitUseCase)
	configService := config_services.NewConfigService(configUseCase)
	pullRequestService := pr_services.NewPullRequestService(pullRequestUseCase)

	cliHandler := cli.NewCLI(commitService, configService, pullRequestService)

	if err := cliHandler.Run(os.Args); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

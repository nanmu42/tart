package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/nanmu42/tart/executor"
	"github.com/nanmu42/tart/network"
	"github.com/nanmu42/tart/runner"

	"go.uber.org/zap"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Listen and run CI jobs",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		logger, err := zap.NewDevelopment()
		if err != nil {
			err = fmt.Errorf("initializing logger: %w", err)
			return
		}

		cfg, err := loadConfig()
		if err != nil {
			err = fmt.Errorf("loading config: %w", err)
			return
		}

		client, err := network.NewClient(network.ClientOpt{
			Endpoint: cfg.GitlabEndpoint,
			Features: executor.SupportFeatures(),
		})
		if err != nil {
			err = fmt.Errorf("initializing Gitlab client: %w", err)
			return
		}

		tart, err := runner.NewRunner(runner.Opt{
			Logger:         logger,
			AccessToken:    cfg.AccessToken,
			Client:         client,
			ExecutorConfig: cfg.Executor,
		})
		if err != nil {
			err = fmt.Errorf("initializing runner: %w", err)
			return
		}

		logger.Info("start to polling new job...")
		for {
			badLuck := runSingle(ctx, tart)
			if badLuck == nil {
				continue
			}
			if errors.Is(badLuck, context.Canceled) || errors.Is(badLuck, context.DeadlineExceeded) {
				logger.Info("received signal, exit.")
				return
			}

			logger.Info("error when polling and running job", zap.Error(badLuck))
		}
	},
}

func runSingle(ctx context.Context, tart *runner.Runner) (err error) {
	job, err := tart.PollNewJob(ctx)
	if err != nil {
		err = fmt.Errorf("polling new job: %w", err)
		return
	}

	err = tart.RunJob(ctx, job)
	if err != nil {
		err = fmt.Errorf("running job: %w", err)
		return
	}

	return
}

package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/nanmu42/tart/executor"
	"github.com/nanmu42/tart/network"
	"github.com/nanmu42/tart/runner"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func init() {
	rootCmd.AddCommand(singleCmd)
}

var singleCmd = &cobra.Command{
	Use:   "single",
	Short: "Listen, wait and run a single CI job, then exit",
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
		job, err := tart.PollNewJob(ctx)
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			logger.Info("received signal, exit.")
			return
		}
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
	},
}

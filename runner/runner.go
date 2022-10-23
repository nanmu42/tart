package runner

import (
	"context"
	"errors"
	"fmt"
	"tart/executor"
	"tart/network"
	"time"

	"go.uber.org/zap"
)

type Opt struct {
	Logger         *zap.Logger
	AccessToken    string
	Client         *network.Client
	ExecutorConfig executor.Config
}

type Runner struct {
	logger         *zap.Logger
	accessToken    string
	client         *network.Client
	executorConfig executor.Config
}

func NewRunner(opt Opt) (runner *Runner, err error) {
	runner = &Runner{
		logger:         opt.Logger,
		accessToken:    opt.AccessToken,
		client:         opt.Client,
		executorConfig: opt.ExecutorConfig,
	}
	return
}

func (r *Runner) PollNewJob(ctx context.Context) (job network.RequestJobResp, err error) {
	const interval = 5 * time.Second
	done := ctx.Done()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			err = ctx.Err()
			return
		case <-ticker.C:
			// relax
		}

		job, err = r.client.RequestJob(ctx, r.accessToken)
		if err == nil {
			r.logger.Info("got new job", zap.Reflect("job", job))
			return
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}

		r.logger.Debug("polling new job...", zap.Error(err), zap.Duration("interval", interval))
	}
}

func (r *Runner) RunJob(ctx context.Context, job network.RequestJobResp) (err error) {
	var result executor.BuildResult

	traceSink, err := network.NewJobTrace(network.JobTraceOpt{
		Logger:   r.logger,
		Client:   r.client,
		JobToken: job.Token,
		JobID:    job.ID,
	})
	if err != nil {
		err = fmt.Errorf("init trace: %w", err)
		return
	}
	defer func() {
		if result.Err != nil {
			_ = traceSink.Fail(ctx, result.ExitCode, result.FailureReason)
			return
		}
		if err != nil {
			_ = traceSink.Fail(ctx, 0, network.FailureReasonRunnerSystemFailure)
			return
		}

		_ = traceSink.Complete(ctx)
	}()

	build, err := executor.NewBuild(executor.BuildOpt{
		Job:        job,
		WorkingDir: "ci-repo",
	})
	if err != nil {
		err = fmt.Errorf("initializing build: %w", err)
		return
	}

	exe, err := executor.NewExecutor(executor.Option{
		Logger:   r.logger,
		Ctx:      ctx,
		Build:    build,
		JobTrace: traceSink,
		Config:   r.executorConfig,
	})
	if err != nil {
		err = fmt.Errorf("initializing executor: %w", err)
		return
	}
	defer exe.Close(ctx)

	err = exe.Prepare(ctx)
	if err != nil {
		err = fmt.Errorf("preparing build: %w", err)
		return
	}

	result = exe.Build()
	err = result.Err
	if err != nil {
		err = fmt.Errorf("running build: %w", err)
		return
	}

	return
}

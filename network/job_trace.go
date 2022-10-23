package network

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

type JobTrace struct {
	logger *zap.Logger
	client *Client

	jobToken string
	jobID    int

	// is the recording of job trace stopped?
	finished chan struct{}

	// protect following fields
	mu sync.Mutex

	sink          *os.File
	checksum      hash.Hash32
	writtenBytes  int
	uploadedBytes int
}

type JobTraceOpt struct {
	Logger *zap.Logger
	Client *Client

	JobToken string
	JobID    int
}

func NewJobTrace(opt JobTraceOpt) (trace *JobTrace, err error) {
	file, err := os.CreateTemp("", "tart-job-log-*.txt")
	if err != nil {
		err = fmt.Errorf("creating temp file: %w", err)
		return
	}

	trace = &JobTrace{
		logger:        opt.Logger,
		client:        opt.Client,
		jobToken:      opt.JobToken,
		jobID:         opt.JobID,
		finished:      make(chan struct{}),
		mu:            sync.Mutex{},
		sink:          file,
		checksum:      crc32.NewIEEE(),
		writtenBytes:  0,
		uploadedBytes: 0,
	}

	go trace.intervalAppendTrace()

	return
}

func (t *JobTrace) Write(p []byte) (n int, err error) {
	select {
	case <-t.finished:
		err = errors.New("trace collecting is finished")
		return
	default:
		// relax
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	n, err = t.sink.Write(p)
	if err != nil {
		err = fmt.Errorf("writing to sink: %w", err)
		return
	}
	_, err = t.checksum.Write(p)
	if err != nil {
		err = fmt.Errorf("writing to crc32: %w", err)
		return
	}

	t.writtenBytes += n

	return
}

func (t *JobTrace) Complete(ctx context.Context) error {
	return t.finish(ctx, finishTraceParam{
		State:         JobStateSuccess,
		ExitCode:      0,
		FailureReason: "",
	})
}

func (t *JobTrace) Fail(ctx context.Context, exitCode int, reason FailureReason) error {
	return t.finish(ctx, finishTraceParam{
		State:         JobStateFailed,
		ExitCode:      exitCode,
		FailureReason: reason,
	})
}

type finishTraceParam struct {
	State         JobState
	ExitCode      int
	FailureReason FailureReason
}

func (t *JobTrace) finish(ctx context.Context, param finishTraceParam) (err error) {
	select {
	case <-t.finished:
		err = errors.New("job trace has been finished")
		return
	default:
		// relax
	}

	close(t.finished)

	err = t.appendingUpload()
	if err != nil {
		err = fmt.Errorf("flushing trace log: %w", err)
		return
	}

	err = t.client.UpdateJob(ctx, UpdateJobParam{
		JobToken:      t.jobToken,
		JobID:         t.jobID,
		State:         param.State,
		TraceChecksum: "crc32:" + hex.EncodeToString(t.checksum.Sum(nil)),
		TraceByteSize: t.uploadedBytes,
		ExitCode:      param.ExitCode,
		FailureReason: param.FailureReason,
	})
	if err != nil {
		err = fmt.Errorf("updatig job status: %w", err)
		return
	}

	err = t.sink.Close()
	if err != nil {
		err = fmt.Errorf("closing temp log file: %w", err)
		return
	}
	_ = os.Remove(t.sink.Name())

	return
}

func (t *JobTrace) appendingUpload() (err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.writtenBytes == 0 || t.uploadedBytes >= t.writtenBytes {
		// nothing to do
		return
	}

	length := t.writtenBytes - t.uploadedBytes

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	nextRangeStart, err := t.client.AppendJobTrace(ctx, AppendJobTraceParam{
		JobToken:      t.jobToken,
		JobID:         t.jobID,
		Reader:        io.NewSectionReader(t.sink, int64(t.uploadedBytes), int64(length)),
		ContentLength: length,
		RangeStart:    t.uploadedBytes,
	})
	if err != nil {
		err = fmt.Errorf("append job trace to Gitlab: %w", err)
		return
	}

	t.uploadedBytes = nextRangeStart

	return
}

func (t *JobTrace) intervalAppendTrace() {
	const (
		defaultPeriod = 10 * time.Second
		retryPeriod   = 3 * time.Second
	)

	var err error

	ticker := time.NewTicker(defaultPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err = t.appendingUpload()
			if err != nil {
				err = fmt.Errorf("appendingUpload: %w", err)
				t.logger.Warn("appending trace failed. Retry in 3 seconds...", zap.Error(err))
				ticker.Reset(retryPeriod)
				continue
			}
			ticker.Reset(defaultPeriod)
		case <-t.finished:
			ticker.Stop()
			return
		}
	}
}

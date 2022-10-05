package network

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"tart/version"

	"go.uber.org/atomic"
)

// Client bridges Tart and Gitlab
type Client struct {
	client *http.Client
	// What feature does the runner support?
	features Features
	// API endpoint, only scheme + host, e.g. https://gitlab.example.com
	endpoint string

	lastUpdateCursor *atomic.String
}

type ClientOpt struct {
	// API endpoint, only scheme + host, e.g. https://gitlab.example.com
	Endpoint string
	// What feature does the runner support?
	Features Features
}

func NewClient(opt ClientOpt) (client *Client, err error) {
	endpoint, err := url.Parse(opt.Endpoint)
	if err != nil {
		err = fmt.Errorf("parsing endpoint: %w", err)
		return
	}
	if endpoint.Scheme == "" {
		err = errors.New("endpoint scheme is empty")
		return
	}
	if endpoint.Scheme != "http" && endpoint.Scheme != "https" {
		err = fmt.Errorf("unexpected endpoint scheme, want http or https, got %q", endpoint.Scheme)
		return
	}
	if endpoint.Host == "" {
		err = errors.New("endpoint host is empty")
		return
	}

	client = &Client{
		client:           &http.Client{},
		features:         opt.Features,
		endpoint:         fmt.Sprintf("%s://%s", endpoint.Scheme, endpoint.Host),
		lastUpdateCursor: atomic.NewString(""),
	}

	return
}

func (c *Client) newRequest(ctx context.Context, method string, path string, body any) (req *http.Request, err error) {
	var bodyReader io.Reader
	if body == nil {
		bodyReader = nil
	} else if bodyAsReader, ok := body.(io.Reader); ok {
		bodyReader = bodyAsReader
	} else {
		var marshaled []byte
		marshaled, err = json.Marshal(body)
		if err != nil {
			err = fmt.Errorf("marshaling body into JSON: %w", err)
			return
		}
		bodyReader = bytes.NewReader(marshaled)
	}

	req, err = http.NewRequest(method, c.endpoint+path, bodyReader)
	if err != nil {
		err = fmt.Errorf("forging request: %w", err)
		return
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", version.FullName)

	return
}

func (c *Client) info() Info {
	return Info{
		Architecture: "amd64",
		// let's pretend we are shell
		Executor: "shell",
		Shell:    "bash",
		Features: c.features,
		Name:     version.Name,
		Platform: "linux",
		Revision: version.Revision,
		Version:  version.FullName,
	}
}

func drainAndCloseBody(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func isResponseOK(resp *http.Response) (err error) {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return
	}

	body := make([]byte, 256)
	_, _ = io.ReadFull(resp.Body, body)
	err = fmt.Errorf("API responds with code %d(%s), body: %q", resp.StatusCode, resp.Status, body)
	return
}

func unmarshalJSON(dest any, body io.Reader) (err error) {
	decoder := json.NewDecoder(body)
	err = decoder.Decode(dest)
	if err != nil {
		err = fmt.Errorf("unmarshaling JSON: %w", err)
		return
	}

	return
}

type RegisterParam struct {
	// Registration token
	Token string
	// Runner's description
	Description string
}

// Register Registers a new Runner and obtains its access token.
func (c *Client) Register(ctx context.Context, param RegisterParam) (accessToken string, err error) {
	body := RegisterReq{
		Token:           param.Token,
		Description:     param.Description,
		Info:            c.info(),
		Locked:          false,
		MaintenanceNote: "Tart is an educational purpose toy CI runner.",
		Paused:          false,
		RunUntagged:     true,
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/api/v4/runners", body)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		err = fmt.Errorf("calling Gitlab API: %w", err)
		return
	}
	defer drainAndCloseBody(resp)

	err = isResponseOK(resp)
	if err != nil {
		return
	}

	var respBody RegisterResp
	err = unmarshalJSON(&respBody, resp.Body)
	if err != nil {
		return
	}

	accessToken = respBody.Token
	return
}

var ErrNoJobAvailable = errors.New("no job available")

func (c *Client) RequestJob(ctx context.Context, accessToken string) (job RequestJobResp, err error) {
	reqBody := RequestJobReq{
		Info:       c.info(),
		LastUpdate: c.lastUpdateCursor.Load(),
		Token:      accessToken,
	}

	req, err := c.newRequest(ctx, http.MethodPost, "/api/v4/jobs/request", reqBody)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		err = fmt.Errorf("calling Gitlab API: %w", err)
		return
	}
	defer drainAndCloseBody(resp)

	err = isResponseOK(resp)
	if err != nil {
		return
	}

	lastUpdateCursor := resp.Header.Get("X-Gitlab-Last-Update")
	if lastUpdateCursor != "" {
		c.lastUpdateCursor.Store(lastUpdateCursor)
	}

	if resp.StatusCode == http.StatusNoContent {
		err = ErrNoJobAvailable
		return
	}

	err = unmarshalJSON(&job, resp.Body)
	if err != nil {
		return
	}

	return
}

type UpdateJobParam struct {
	// Job's authentication token
	JobToken string
	// Job's ID
	JobID int
	// Job's status: success, failed
	State JobState
	// Job's trace CRC32 checksum
	TraceChecksum string
	// Job's trace size in bytes
	TraceByteSize int
	// Job's exit code
	ExitCode int
	// Job's failure_reason
	FailureReason FailureReason
}

func (c *Client) UpdateJob(ctx context.Context, param UpdateJobParam) (err error) {
	reqBody := UpdateJobReq{
		Checksum:      param.TraceChecksum,
		ExitCode:      param.ExitCode,
		FailureReason: param.FailureReason,
		Info:          c.info(),
		Output: TraceSummary{
			ByteSize: param.TraceByteSize,
			Checksum: param.TraceChecksum,
		},
		State: param.State,
		Token: param.JobToken,
	}

	req, err := c.newRequest(ctx, http.MethodPut, "/api/v4/jobs/"+strconv.Itoa(param.JobID), reqBody)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		err = fmt.Errorf("calling Gitlab API: %w", err)
		return
	}
	defer drainAndCloseBody(resp)

	err = isResponseOK(resp)
	if err != nil {
		return
	}

	return
}

type AppendJobTraceParam struct {
	// Job's authentication token
	JobToken string
	// Job's ID
	JobID int
	// Job trace source
	Reader io.Reader
	// Size of trace from Reader, in bytes
	ContentLength int
	// Range
	RangeStart int
}

func (c *Client) AppendJobTrace(ctx context.Context, param AppendJobTraceParam) (nextRangeStart int, err error) {
	if param.ContentLength <= 0 {
		err = fmt.Errorf("contentLength must be positive, got %d", param.ContentLength)
		return
	}

	req, err := c.newRequest(ctx, http.MethodPatch, fmt.Sprintf("/api/v4/jobs/%d/trace", param.JobID), param.Reader)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Job-Token", param.JobToken)
	req.Header.Set("Content-Length", strconv.Itoa(param.ContentLength))
	// Both left and right are zero-indexed & inclusive.
	// ref: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Content-Range
	req.Header.Set("Content-Range", fmt.Sprintf("%d-%d", param.RangeStart, param.RangeStart+param.ContentLength-1))

	resp, err := c.client.Do(req)
	if err != nil {
		err = fmt.Errorf("calling Gitlab API: %w", err)
		return
	}
	defer drainAndCloseBody(resp)

	err = isResponseOK(resp)
	if err != nil {
		return
	}

	nextRangeStart = param.RangeStart + param.ContentLength
	return
}

package network

// Features describes the runner's abilities.
// Since Tart is a toy runner, a very limited set of features are supported.
type Features struct {
	Shared                  bool `json:"shared"`
	MultiBuildSteps         bool `json:"multi_build_steps"`
	Cancelable              bool `json:"cancelable"`
	ReturnExitCode          bool `json:"return_exit_code"`
	Variables               bool `json:"variables"`
	RawVariables            bool `json:"raw_variables"`
	Artifacts               bool `json:"artifacts"`
	UploadMultipleArtifacts bool `json:"upload_multiple_artifacts"`
	UploadRawArtifacts      bool `json:"upload_raw_artifacts"`
	ArtifactsExclude        bool `json:"artifacts_exclude"`
	TraceReset              bool `json:"trace_reset"`
	TraceChecksum           bool `json:"trace_checksum"`
	TraceSize               bool `json:"trace_size"`
}

type RegisterReq struct {
	// Registration token
	Token string `json:"token"`
	// Runner's description
	Description string `json:"description"`
	// runner meta data
	Info Info `json:"info"`
	// Whether the runner should be locked for current project
	Locked bool `json:"locked"`
	// Runner's maintenance notes
	MaintenanceNote string `json:"maintenance_note"`
	// Whether the runner should ignore new jobs
	Paused bool `json:"paused"`
	// Whether the runner should handle untagged jobs
	RunUntagged bool `json:"run_untagged"`
}

type Info struct {
	// e.g. amd64
	Architecture string `json:"architecture"`
	// e.g. shell
	Executor string `json:"executor,omitempty"`
	// e.g. bash
	Shell string `json:"shell,omitempty"`
	// supported features
	Features Features `json:"features"`
	// e.g. gitlab-runner
	Name string `json:"name"`
	// e.g. linux
	Platform string `json:"platform"`
	// e.g. f98d0f26
	Revision string `json:"revision"`
	// e.g. 15.2.0~beta.60.gf98d0f26
	Version string `json:"version"`
}

type RegisterResp struct {
	// Runner's ID on Gitlab side
	ID int `json:"id"`
	// Runner's authentication token
	Token string `json:"token"`
}

type RequestJobReq struct {
	// runner meta data
	Info Info `json:"info"`
	// runner work queue cursor, for cache purpose
	LastUpdate string `json:"last_update"`
	// Runner's authentication token
	Token string `json:"token"`
}

type RequestJobResp struct {
	ID            int             `json:"id"`
	AllowGitFetch bool            `json:"allow_git_fetch"`
	Artifacts     interface{}     `json:"artifacts"`
	Credentials   []JobCredential `json:"credentials"`
	GitInfo       GitInfo         `json:"git_info"`
	JobInfo       JobInfo         `json:"job_info"`
	Steps         []JobStep       `json:"steps"`
	Token         string          `json:"token"`
	Variables     []JobVariable   `json:"variables"`
}

type JobCredential struct {
	Password string `json:"password"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Username string `json:"username"`
}

type GitInfo struct {
	BeforeSha string   `json:"before_sha"`
	Depth     int      `json:"depth"`
	Ref       string   `json:"ref"`
	RefType   string   `json:"ref_type"`
	RefSpecs  []string `json:"refspecs"`
	RepoURL   string   `json:"repo_url"`
	Sha       string   `json:"sha"`
}

type JobInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ProjectID   int    `json:"project_id"`
	ProjectName string `json:"project_name"`
	Stage       string `json:"stage"`
}

type JobStep struct {
	AllowFailure bool     `json:"allow_failure"`
	Name         string   `json:"name"`
	Script       []string `json:"script"`
	Timeout      int      `json:"timeout"`
	When         string   `json:"when"`
}

type JobVariable struct {
	Key    string `json:"key"`
	Masked bool   `json:"masked"`
	Public bool   `json:"public"`
	Value  string `json:"value"`
}

type JobState string

const (
	JobStateRunning JobState = "running"
	JobStateSuccess JobState = "success"
	JobStateFailed  JobState = "failed"
)

type FailureReason string

const (
	FailureReasonScriptFailure       FailureReason = "script_failure"
	FailureReasonRunnerSystemFailure FailureReason = "runner_system_failure"
	FailureReasonArchivedFailure     FailureReason = "archived_failure"
	FailureReasonJobTimeout          FailureReason = "job_execution_timeout"
	FailureReasonRunnerUnsupported   FailureReason = "runner_unsupported"
)

type UpdateJobReq struct {
	Checksum      string        `json:"checksum"`
	ExitCode      int           `json:"exit_code"`
	FailureReason FailureReason `json:"failure_reason"`
	Info          Info          `json:"info"`
	Output        TraceSummary  `json:"output"`
	State         JobState      `json:"state"`
	Token         string        `json:"token"`
}

type TraceSummary struct {
	ByteSize int    `json:"bytesize"`
	Checksum string `json:"checksum"`
}

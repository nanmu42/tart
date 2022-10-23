package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"tart/network"
	"tart/rootfs"
	"tart/version"
	"time"

	"go.uber.org/zap"

	"github.com/fatih/color"

	"golang.org/x/crypto/ssh"

	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"

	"github.com/firecracker-microvm/firecracker-go-sdk"
)

type Config struct {
	// path to linux kernel file
	KernelPath string `comment:"path to linux kernel file"`
	// path to RootFS file
	RootFSPath string `comment:"path to RootFS file"`

	// IP address of Firecracker microVM
	IP string `comment:"IP address of Firecracker microVM"`
	// Gateway IP address, normally is the tap address
	GatewayIP string `comment:"Gateway IP address, normally is the tap address"`
	// Netmask like 255.255.255.0
	Netmask string `comment:"Netmask like 255.255.255.0"`

	// Tap device name like tap0
	TapDevice string `comment:"Tap device name like tap0"`
	// microVM tap MAC address
	TapMac string `comment:"microVM tap MAC address"`
}

func SupportFeatures() network.Features {
	return network.Features{
		Shared:                  true,
		MultiBuildSteps:         true,
		Cancelable:              true,
		ReturnExitCode:          true,
		Variables:               true,
		RawVariables:            true,
		Artifacts:               true,
		UploadMultipleArtifacts: true,
		UploadRawArtifacts:      true,
		ArtifactsExclude:        true,
		TraceReset:              true,
		TraceChecksum:           true,
		TraceSize:               true,
	}
}

func (c Config) Validate() (err error) {
	if c.KernelPath == "" {
		err = errors.New("kernel path is required")
		return
	}
	if c.RootFSPath == "" {
		err = errors.New("rootFS path is required")
		return
	}
	if c.IP == "" {
		err = errors.New("ip is required")
		return
	}
	if c.GatewayIP == "" {
		err = errors.New("gatewayIP is required")
		return
	}
	if c.Netmask == "" {
		err = errors.New("netmask is required")
		return
	}
	if c.TapDevice == "" {
		err = errors.New("tap device is required")
		return
	}
	if c.TapMac == "" {
		err = errors.New("tap MAC is required")
		return
	}

	return
}

type Option struct {
	Logger *zap.Logger
	// The context must not be cancelled while the microVM is running.
	Ctx      context.Context
	Build    *Build
	JobTrace *network.JobTrace

	Config
}

type Executor struct {
	logger *zap.Logger
	// The context must not be cancelled while the microVM is running.
	ctx    context.Context
	build  *Build
	config Config

	logSink        io.Writer
	socketFilePath string
	tempRootFS     *os.File
	machine        *firecracker.Machine
	ssh            *ssh.Client
}

func NewExecutor(opt Option) (e *Executor, err error) {
	if opt.Logger == nil {
		err = errors.New("logger must be non-nil")
		return
	}
	if opt.Ctx == nil {
		err = errors.New("ctx must be non-nil")
		return
	}
	if opt.Build == nil {
		err = errors.New("build is required")
		return
	}
	if opt.JobTrace == nil {
		err = errors.New("job trace is required")
		return
	}

	err = opt.Config.Validate()
	if err != nil {
		err = fmt.Errorf("validating config: %w", err)
		return
	}

	logger := opt.Logger.With(zap.Int("jobId", opt.Build.job.ID))

	e = &Executor{
		logger:     logger,
		ctx:        opt.Ctx,
		build:      opt.Build,
		config:     opt.Config,
		logSink:    opt.JobTrace,
		tempRootFS: nil,
		machine:    nil,
		ssh:        nil,
	}

	rootFSOrigin, err := os.Open(e.config.RootFSPath)
	if err != nil {
		err = fmt.Errorf("open original RootFS file: %w", err)
		return
	}
	defer rootFSOrigin.Close()

	e.tempRootFS, err = os.CreateTemp("", "tart-rootfs-*.ext4")
	if err != nil {
		err = fmt.Errorf("creating temp rootFS: %w", err)
		return
	}
	_, err = io.Copy(e.tempRootFS, rootFSOrigin)
	if err != nil {
		err = fmt.Errorf("clone rootFS: %w", err)
		return
	}
	err = e.tempRootFS.Sync()
	if err != nil {
		err = fmt.Errorf("file system sync on rootFS: %w", err)
		return
	}

	return
}

type freezeReader struct{}

func (f freezeReader) Read(p []byte) (n int, err error) {
	// freezes here
	select {}
}

// Prepare start the VM, clones the repo.
func (e *Executor) Prepare(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			e.logger.Debug("Build failed during preparing", zap.Error(err))
			_ = e.redLine("Build failed during preparing: %s", err)
		}
	}()

	err = e.yellowLine("Running with %s on %s\n", version.FullName, e.config.IP)
	if err != nil {
		return
	}

	e.logger.Debug("Spinning up microVM...")
	err = e.blueLine("Spinning up microVM...")
	if err != nil {
		return
	}

	e.socketFilePath = fmt.Sprintf("/tmp/tart-firecracker-%d.socket", time.Now().UnixNano())
	cmd := firecracker.VMCommandBuilder{}.
		WithStdin(freezeReader{}).
		WithStdout(io.Discard).
		WithStderr(io.Discard).
		WithSocketPath(e.socketFilePath).
		Build(ctx)

	machine, err := firecracker.NewMachine(ctx, firecracker.Config{
		SocketPath:      e.socketFilePath,
		KernelImagePath: e.config.KernelPath,
		KernelArgs:      e.kernelArgs(),
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("1"),
				IsReadOnly:   firecracker.Bool(false),
				IsRootDevice: firecracker.Bool(true),
				PathOnHost:   firecracker.String(e.tempRootFS.Name()),
			},
		},
		NetworkInterfaces: []firecracker.NetworkInterface{
			{
				StaticConfiguration: &firecracker.StaticNetworkConfiguration{
					MacAddress:  e.config.TapMac,
					HostDevName: e.config.TapDevice,
				},
			},
		},
		MachineCfg: models.MachineConfiguration{
			MemSizeMib: firecracker.Int64(1024),
			VcpuCount:  firecracker.Int64(2),
		},
	}, firecracker.WithProcessRunner(cmd))
	if err != nil {
		err = fmt.Errorf("init firecracker machine: %w", err)
		return
	}

	e.logger.Debug("MicroVM is initialized, starting...", zap.String("VMID", machine.Cfg.VMID))
	err = e.greenLine("MicroVM %s is initialized, starting...", machine.Cfg.VMID)
	if err != nil {
		return
	}

	err = machine.Start(e.ctx)
	if err != nil {
		err = fmt.Errorf("starting the VM: %w", err)
		return
	}
	e.machine = machine

	e.logger.Debug("MicroVM started, connecting...")
	err = e.greenLine("MicroVM started, connecting...")
	if err != nil {
		return
	}

	// retry until timeout since the VM is booting and may not be ready
	sshCtx, cancelSSHCtx := context.WithTimeout(ctx, 10*time.Second)
	defer cancelSSHCtx()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

TRYSSH:
	for {
		select {
		case <-sshCtx.Done():
			err = fmt.Errorf("waiting for SSH connection to VM: %w", err)
			return
		case <-ticker.C:
			e.ssh, err = e.dialSSH()
			if err == nil {
				break TRYSSH
			}
			e.logger.Debug("trying to establishing SSH connection to VM", zap.Error(err))
		}
	}

	session, err := e.ssh.NewSession()
	if err != nil {
		err = fmt.Errorf("init ssh session: %w", err)
		return
	}
	defer session.Close()

	session.Stdout = e.logSink
	session.Stderr = e.logSink

	err = e.greenLine("MicroVM connected, cloning repo and checking out...")
	if err != nil {
		return
	}

	var buf bytes.Buffer
	err = e.build.PrepareScript(&buf)
	if err != nil {
		err = fmt.Errorf("forging prepare script: %w", err)
		return
	}

	e.logger.Debug("MicroVM connected, cloning repo and checking out...", zap.String("script", buf.String()))
	err = session.Start(buf.String())
	if err != nil {
		err = fmt.Errorf("sending prepare script over SSH: %w", err)
		return
	}

	err = runUntilTimeout(e.build.Timeout(), session.Wait)
	if err != nil {
		err = fmt.Errorf("running prepare script over SSH: %w", err)
		return
	}

	err = e.greenLine("Repo cloned and checked out.")
	if err != nil {
		return
	}

	return
}

type BuildResult struct {
	Err           error
	ExitCode      int
	FailureReason network.FailureReason
}

// Build runs the build and returns encountered error.
func (e *Executor) Build() (result BuildResult) {
	var err error

	defer func() {
		if err != nil {
			e.logger.Debug("Build failed", zap.Error(err))
			_ = e.redLine("Build failed: %s", err)

			if result.Err == nil {
				result.Err = err
			}
			if result.FailureReason == "" {
				result.FailureReason = network.FailureReasonRunnerSystemFailure
			}
		}
	}()

	err = e.blueLine("build phase starting...")
	if err != nil {
		return
	}

	session, err := e.ssh.NewSession()
	if err != nil {
		err = fmt.Errorf("init ssh session: %w", err)
		return
	}
	defer session.Close()

	session.Stdout = e.logSink
	session.Stderr = e.logSink

	var buf bytes.Buffer
	err = e.build.BuildScript(&buf)
	if err != nil {
		err = fmt.Errorf("forging build script: %w", err)
		return
	}

	e.logger.Debug("excuting build script", zap.String("script", buf.String()))
	err = session.Start(buf.String())
	if err != nil {
		err = fmt.Errorf("sending build script over SSH: %w", err)
		return
	}

	err = runUntilTimeout(e.build.Timeout(), session.Wait)
	switch typed := err.(type) {
	case *ssh.ExitError:
		result = BuildResult{
			Err:           typed,
			ExitCode:      typed.ExitStatus(),
			FailureReason: network.FailureReasonScriptFailure,
		}

		return
	case *ssh.ExitMissingError:
		result = BuildResult{
			Err:           err,
			ExitCode:      0,
			FailureReason: network.FailureReasonScriptFailure,
		}
	}
	if err != nil {
		err = fmt.Errorf("running build script over SSH: %w", err)
		return
	}

	e.logger.Debug("Job succeeded")
	err = e.greenLine("Job succeeded")
	if err != nil {
		return
	}

	return
}

func (e *Executor) Close(ctx context.Context) (err error) {
	if e.ssh != nil {
		_ = e.ssh.Close()
	}

	if e.machine != nil {
		err = e.machine.Shutdown(ctx)
		if err != nil {
			err = e.machine.StopVMM()
		}
		if err != nil {
			err = fmt.Errorf("stopping VM: %w", err)
			return
		}
	}
	if e.socketFilePath != "" {
		_ = os.Remove(e.socketFilePath)
	}

	tempRootFSPath := e.tempRootFS.Name()
	_ = e.tempRootFS.Close()

	_ = os.Remove(tempRootFSPath)

	return
}

func (e *Executor) kernelArgs() string {
	return "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules random.trust_cpu=on " +
		fmt.Sprintf("ip=%s::%s:%s::eth0:off", e.config.IP, e.config.GatewayIP, e.config.Netmask)
}

func (e *Executor) dialSSH() (client *ssh.Client, err error) {
	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(rootfs.SSHSigner),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err = ssh.Dial("tcp", e.config.IP+":22", config)
	if err != nil {
		err = fmt.Errorf("dialing ssh: %w", err)
		return
	}

	return
}

func (e *Executor) redLine(format string, args ...any) (err error) {
	_, err = io.WriteString(e.logSink, color.HiRedString(format+"\n", args...))
	if err != nil {
		err = fmt.Errorf("print red line: %w", err)
		return
	}

	return
}

func (e *Executor) yellowLine(format string, args ...any) (err error) {
	_, err = io.WriteString(e.logSink, color.HiYellowString(format+"\n", args...))
	if err != nil {
		err = fmt.Errorf("print yellow line: %w", err)
		return
	}

	return
}

func (e *Executor) blueLine(format string, args ...any) (err error) {
	_, err = io.WriteString(e.logSink, color.HiBlueString(format+"\n", args...))
	if err != nil {
		err = fmt.Errorf("print blue line: %w", err)
		return
	}

	return
}

func (e *Executor) greenLine(format string, args ...any) (err error) {
	_, err = io.WriteString(e.logSink, color.HiGreenString(format+"\n", args...))
	if err != nil {
		err = fmt.Errorf("print green line: %w", err)
		return
	}

	return
}

func (e *Executor) line(format string, args ...any) (err error) {
	_, err = fmt.Fprintf(e.logSink, format+"\n", args...)
	if err != nil {
		err = fmt.Errorf("print line: %w", err)
		return
	}

	return
}

func runUntilTimeout(timeout time.Duration, run func() error) (err error) {
	channel := make(chan error, 1)
	timer := time.NewTicker(timeout)
	defer timer.Stop()

	go func() {
		channel <- run()
	}()

	select {
	case err = <-channel:
		return
	case <-timer.C:
		err = fmt.Errorf("execution timed out after %s", timeout)
	}

	return
}

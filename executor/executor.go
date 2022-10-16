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
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"

	"github.com/firecracker-microvm/firecracker-go-sdk"
)

type Config struct {
	KernelPath string
	RootFSPath string

	IP        string
	GatewayIP string
	Netmask   string

	TapDevice string
	TapMac    string
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
	// The context must not be cancelled while the microVM is running.
	Ctx      context.Context
	Build    *Build
	JobTrace *network.JobTrace

	Config
}

type Executor struct {
	// The context must not be cancelled while the microVM is running.
	ctx    context.Context
	build  *Build
	config Config

	logSink    io.Writer
	tempRootFS *os.File
	machine    *firecracker.Machine
	ssh        *ssh.Client
}

func NewExecutor(opt Option) (e *Executor, err error) {
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

	e = &Executor{
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

	return
}

// Prepare start the VM, clones the repo.
func (e *Executor) Prepare(ctx context.Context) (err error) {
	machine, err := firecracker.NewMachine(ctx, firecracker.Config{
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
	})
	if err != nil {
		err = fmt.Errorf("init firecracker machine: %w", err)
		return
	}

	err = machine.Start(e.ctx)
	if err != nil {
		err = fmt.Errorf("starting the VM: %w", err)
		return
	}
	e.machine = machine

	e.ssh, err = e.dialSSH()
	if err != nil {
		err = fmt.Errorf("establish SSH connection to VM: %w", err)
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
	err = e.build.PrepareScript(&buf)
	if err != nil {
		err = fmt.Errorf("forging prepare script: %w", err)
		return
	}
	err = session.Run(buf.String())
	if err != nil {
		err = fmt.Errorf("sending prepare script over SSH: %w", err)
		return
	}

	err = session.Wait()
	if err != nil {
		err = fmt.Errorf("running prepare script over SSH: %w", err)
		return
	}

	return
}

func (e *Executor) Build(ctx context.Context) (err error) {
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
	err = session.Run(buf.String())
	if err != nil {
		err = fmt.Errorf("sending build script over SSH: %w", err)
		return
	}

	err = session.Wait()
	if err != nil {
		err = fmt.Errorf("running build script over SSH: %w", err)
		return
	}

	return
}

func (e *Executor) Close(ctx context.Context) (err error) {
	_ = e.ssh.Close()

	err = e.machine.Shutdown(ctx)
	if err != nil {
		err = e.machine.StopVMM()
	}
	if err != nil {
		err = fmt.Errorf("stopping VM: %w", err)
		return
	}

	tempRootFSPath := e.tempRootFS.Name()
	_ = e.tempRootFS.Close()

	err = os.Remove(tempRootFSPath)
	if err != nil {
		err = fmt.Errorf("removing temp rootFS file: %w", err)
		return
	}

	return
}

func (e *Executor) kernelArgs() string {
	return "ro console=ttyS0 noapic reboot=k panic=1 pci=off nomodules " +
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

	client, err = ssh.Dial("tcp", e.config.IP, config)
	if err != nil {
		err = fmt.Errorf("dialing ssh: %w", err)
		return
	}

	return
}

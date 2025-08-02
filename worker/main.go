package main 

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/google/uuid"
)

type SimplifiedVM struct {
	machine *firecracker.Machine
	socket  string
	stdout  chan string
	stderr  chan string
}

func createVM(ctx context.Context, kernelPath, rootfsPath string) (*SimplifiedVM, error) {
	vmID := uuid.New().String()
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.sock", vmID))

	stdout := make(chan string, 100)
	stderr := make(chan string, 100)

	cfg := firecracker.Config{
		SocketPath:      socketPath,
		KernelImagePath: kernelPath,
		KernelArgs:      "console=ttyS0 reboot=k panic=1 init=/bin/sh root=/dev/vda rw",
		Drives: []models.Drive{
			{
				DriveID:      firecracker.String("1"),
				PathOnHost:   firecracker.String(rootfsPath),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
			},
		},
		NetworkInterfaces: []firecracker.NetworkInterface{
			{
				StaticConfiguration: &firecracker.StaticNetworkConfiguration{
					MacAddress:  "AA:FC:00:00:00:01",
					HostDevName: "tap0",
				},
			},
		},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(1),
			MemSizeMib: firecracker.Int64(128),
		},
		ForwardSignals: []os.Signal{},
		LogLevel:       "Debug",
		LogPath:        filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.log", vmID)),
		MetricsPath:    filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s-metrics", vmID)),
	}

	stdoutFile, err := os.Create(filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.stdout", vmID)))
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout file: %v", err)
	}

	stderrFile, err := os.Create(filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.stderr", vmID)))
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr file: %v", err)
	}

	cmd := firecracker.VMCommandBuilder{}.
		WithBin("firecracker").
		WithSocketPath(socketPath).
		WithStdin(os.Stdin).
		WithStdout(stdoutFile).
		WithStderr(stderrFile).
		Build(ctx)

	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))
	if err != nil {
		return nil, fmt.Errorf("failed to create machine: %v", err)
	}

	return &SimplifiedVM{
		machine: machine,
		socket:  socketPath,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

func (vm *SimplifiedVM) start(ctx context.Context) error {
	if err := vm.machine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start machine: %v", err)
	}

	go func() {
		for {
			select {
			case line := <-vm.stdout:
				if line != "" {
					vm.stdout <- line
				}
			case line := <-vm.stderr:
				if line != "" {
					vm.stderr <- line
				}
			case <-ctx.Done():
				close(vm.stdout)
				close(vm.stderr)
				return
			}
		}
	}()

	return nil
}

func (vm *SimplifiedVM) stop(ctx context.Context) error {
	if err := vm.machine.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop machine: %v", err)
	}

	if err := os.Remove(vm.socket); err != nil {
		log.Printf("failed to remove socket file: %v", err)
	}
	return nil
}

func main() {
	kernelPath := "/var/lib/firecracker/vmlinux"
	rootfsPath := "/var/lib/firecracker/debian_firecracker.ext4"
	pidFile := "/tmp/firecracker.pid"
	socketFile := "/tmp/firecracker.sock"

	// Save PID
	pid := os.Getpid()
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		log.Fatalf("Failed to write PID file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	vm, err := createVM(ctx, kernelPath, rootfsPath)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	// Save socket path
	if err := os.WriteFile(socketFile, []byte(vm.socket), 0644); err != nil {
		log.Fatalf("Failed to write socket file: %v", err)
	}

	if err := vm.start(ctx); err != nil {
		log.Fatalf("Failed to start VM: %v", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start goroutine for VM output
	go func() {
		for {
			select {
			case line := <-vm.stdout:
				fmt.Printf("VM stdout: %s\n", line)
			case line := <-vm.stderr:
				fmt.Printf("VM stderr: %s\n", line)
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Printf("VM started successfully. PID: %d, Socket: %s\n", pid, vm.socket)
	log.Println("To stop the VM, run: kill $(cat /tmp/firecracker.pid)")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down VM...")

	// Cleanup files
	os.Remove(pidFile)
	os.Remove(socketFile)

	if err := vm.stop(ctx); err != nil {
		log.Fatalf("Failed to stop VM: %v", err)
	}
	log.Println("VM stopped successfully")
}

package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"log"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/google/uuid"
)

type VirtualMachine struct {
	machine *firecracker.Machine
	socket  string
}

func CreateVM(ctx context.Context, kernelPath, rootfsPath string, idx int) (*VirtualMachine, error) {
	vmID := uuid.New().String()
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.sock", vmID))

	cfg := firecracker.Config{
		SocketPath: socketPath,
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
					MacAddress:  fmt.Sprintf("AA:FC:00:00:00:0%d", idx),
					HostDevName: fmt.Sprintf("tap%d", idx),
				},
			},
		},
		MachineCfg: models.MachineConfiguration{
			VcpuCount:  firecracker.Int64(1),
			MemSizeMib: firecracker.Int64(256),
		},
		ForwardSignals: []os.Signal{},
		LogLevel:       "Debug",
		LogPath:        filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.log", vmID)),
		MetricsPath:    filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s-metrics", vmID)),
	}

	cmd := firecracker.VMCommandBuilder{}.
			WithBin("firecracker").
			WithSocketPath(socketPath).
			Build(ctx)
	
	machine, err := firecracker.NewMachine(ctx, cfg, firecracker.WithProcessRunner(cmd))

	if err != nil {
		return nil, fmt.Errorf("failed to create machine %v", err)
	}

	return &VirtualMachine{
		machine: machine,
		socket: socketPath,
	}, nil
}

func (vm *VirtualMachine) Start(ctx context.Context) error{
	if err := vm.machine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start machine: %v", err)
	}

	return nil
}

func (vm *VirtualMachine) Stop(ctx context.Context) error {
	if err := vm.machine.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop machine: %v", err)
	}

	if err := os.Remove(vm.socket); err != nil {
		log.Printf("failed to remove socket file: %v", err)
	}
	return nil
}

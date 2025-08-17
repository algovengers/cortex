package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/google/uuid"
)

const maxVms = 4

var (
	vmIndices = make(map[int]bool)
	mu        sync.Mutex
)

type VirtualMachine struct {
	machine *firecracker.Machine
	socket  string
	index   int
}

func InitVms() {
	for i := 0; i < maxVms; i++ {
		vmIndices[i] = false
	}
}

func occupyVM(ctx context.Context) int {
	mu.Lock()
	defer mu.Unlock()
	for i := 0; i < maxVms; i++ {
		if !vmIndices[i] {
			vmIndices[i] = true
			return i
		}
	}

	return -1
}

func releaseVM(idx int) {
	mu.Lock()
	defer mu.Unlock()
	vmIndices[idx] = false
}

func CreateVM(ctx context.Context) (*VirtualMachine, error) {
	kernelPath := os.Getenv("FIRECRACKER_KERNEL_PATH")
	rootfsPath := os.Getenv("FIRECRACKER_ROOTFS_PATH")

	idx := occupyVM(ctx)
	if idx == -1 {
		return nil, fmt.Errorf("no vms available right now")
	}

	defer func() {
		if r := recover(); r != nil {
			releaseVM(idx)
			panic(r)
		}
	}()

	vmID := uuid.New().String()
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.sock", vmID))

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
		socket:  socketPath,
		index:   idx,
	}, nil
}

func (vm *VirtualMachine) Start(ctx context.Context) error {
	if err := vm.machine.Start(ctx); err != nil {
		releaseVM(vm.index)
		return fmt.Errorf("failed to start machine: %v", err)
	}

	return nil
}

func (vm *VirtualMachine) Stop(ctx context.Context) error {
	if err := vm.machine.Shutdown(ctx); err != nil {
		releaseVM(vm.index)
		return fmt.Errorf("failed to stop machine: %v", err)
	}

	if err := os.Remove(vm.socket); err != nil {
		log.Printf("failed to remove socket file: %v", err)
	}
	releaseVM(vm.index)
	return nil
}

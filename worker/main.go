package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/firecracker-microvm/firecracker-go-sdk"
	"github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	"github.com/google/uuid"
)

type VM struct {
	machine *firecracker.Machine
	socket  string
	stdout  chan string
	stdin   chan string
}

func createVM(ctx context.Context, kernelPath, rootFsPath string) (*Vm, error) {
	vmID := uuid.New().String()
	socketPath := filepath.Join(os.TempDir(), fmt.Sprintf("firecracker-%s.sock", vmID))

	stdout := make(chan string, 100)
	stdin := make(chan string, 100)

	cfg := firecracker.Config{
		SocketPath: socketPath,
		KernelImagePath: kernelPath,
		KernelArgs:      "console=ttyS0 reboot=k panic=1 init=/bin/sh root=/dev/vda rw",
		Drives: []models.Drive{
			{
				DriveID: firecracker.String("1"),
			}
		},
	}
}

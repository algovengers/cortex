package worker

import "context"

func StartJob(ctx context.Context) error {
	// Start a VM
	vm, err := CreateVM(ctx)

	if err != nil {
		return err
	}

	err = vm.Start(ctx)
	if err != nil {
		return err
	}

	// Setup scripts within a vm

	// Start an agent within the VM

	// return ack


	return nil
}

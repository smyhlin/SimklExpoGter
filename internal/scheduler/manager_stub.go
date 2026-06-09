//go:build !windows && !linux

package scheduler

import "errors"

type unsupportedManager struct{}

func newManager() Manager {
	return unsupportedManager{}
}

func (unsupportedManager) Supported() bool {
	return false
}

func (unsupportedManager) Sync(Config) (State, error) {
	return State{Supported: false}, errors.New("recurring scheduling is not supported on this platform")
}

func (unsupportedManager) Remove(string) error {
	return nil
}

func (unsupportedManager) Query(taskName string) (State, error) {
	return State{
		Supported: false,
		TaskName:  taskName,
		Message:   "Recurring scheduling is not supported on this platform.",
	}, nil
}

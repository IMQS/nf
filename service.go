package nf

import (
	"os"

	"github.com/IMQS/gowinsvc/service"
)

// IsRunningInContainer returns true if we're running inside a docker container.
func IsRunningInContainer() bool {
	_, err := os.Stat("/.dockerenv")
	return err == nil

}

// RunService runs 'run' as a service on Windows, but if that fails, then falls back to running in the foreground.
func RunService(run func()) {
	if !service.RunAsService(run) {
		// Run in the foreground
		run()
	}
}

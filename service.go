package nf

import (
	"github.com/IMQS/gowinsvc/service"
)

// RunService runs 'run' as a service on Windows, but if that fails, then falls back to running in the foreground.
func RunService(run func()) {
	if !service.RunAsService(run) {
		// Run in the foreground
		run()
	}
}

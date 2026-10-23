package worker

import (
	"context"
)

// Worker interface defines the common behavior for all workers
type Worker interface {
	// Name returns the unique name of the worker
	Name() string

	// Start begins the worker's operation. It should be non-blocking and run in the background
	Start(ctx context.Context) error

	// Stop gracefully stops the worker and cleans up resources
	Stop() error
}

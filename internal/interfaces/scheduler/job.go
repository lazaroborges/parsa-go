package scheduler

import "context"

// Job represents a unit of work that can be executed by the worker pool.
// This interface allows for extensibility - different types of jobs can be
// implemented (e.g., sync jobs, cleanup jobs, notification jobs).
type Job interface {
	// Execute runs the job with the given context.
	// Context should be respected for cancellation and timeouts.
	Execute(ctx context.Context) error

	// UserID returns the user ID associated with this job.
	// This is useful for logging and tracking which user's data is being processed.
	UserID() string

	// Description returns a human-readable description of the job.
	// Used for logging purposes.
	Description() string
}

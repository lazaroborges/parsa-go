package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

var (
	jobTracer          = otel.Tracer("parsa/scheduler")
	jobMeter           = otel.Meter("parsa/scheduler")
	jobDuration, _     = jobMeter.Float64Histogram("scheduler.job.duration", metric.WithDescription("Job execution duration in seconds"), metric.WithUnit("s"))
	jobTotal, _        = jobMeter.Int64Counter("scheduler.job.total", metric.WithDescription("Total jobs executed by status"))
	jobQueueDropped, _ = jobMeter.Int64Counter("scheduler.job.queue_dropped", metric.WithDescription("Jobs dropped due to full queue"))
)

// WorkerPool manages a pool of concurrent workers that process jobs.
// It demonstrates Go's concurrency primitives: goroutines, channels,
// WaitGroups, and context-based cancellation.
type WorkerPool struct {
	workerCount int
	jobDelay    time.Duration
	jobs        chan Job
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewWorkerPool creates a new worker pool with the specified configuration.
// workerCount: number of concurrent workers (goroutines)
// jobDelay: delay between processing jobs (for rate limiting)
// queueSize: buffer size for the job channel
func NewWorkerPool(workerCount int, jobDelay time.Duration, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workerCount: workerCount,
		jobDelay:    jobDelay,
		jobs:        make(chan Job, queueSize),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start launches the worker goroutines.
// Each worker runs in its own goroutine and processes jobs from the channel.
func (wp *WorkerPool) Start() {
	log.Printf("Starting worker pool with %d workers", wp.workerCount)

	for i := 1; i <= wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker(i)
	}
}

// worker is the main loop for each worker goroutine.
// It continuously processes jobs from the channel until shutdown.
// Demonstrates: goroutines, channels, select with context, WaitGroups.
func (wp *WorkerPool) worker(id int) {
	defer wp.wg.Done()

	log.Printf("Worker %d started", id)

	for {
		select {
		case <-wp.ctx.Done():
			// Context cancelled - graceful shutdown
			log.Printf("Worker %d shutting down", id)
			return

		case job, ok := <-wp.jobs:
			if !ok {
				// Channel closed - no more jobs
				log.Printf("Worker %d: job channel closed", id)
				return
			}

			// Process the job
			wp.processJob(id, job)

			// Apply delay to avoid rate limiting (if configured)
			if wp.jobDelay > 0 {
				select {
				case <-time.After(wp.jobDelay):
					// Delay completed
				case <-wp.ctx.Done():
					// Context cancelled during delay
					log.Printf("Worker %d shutting down during delay", id)
					return
				}
			}
		}
	}
}

// processJob executes a single job with error handling, logging, and telemetry.
func (wp *WorkerPool) processJob(workerID int, job Job) {
	log.Printf("Worker %d: Processing %s for user %s", workerID, job.Description(), job.UserID())

	// Create a timeout context for the job execution
	ctx, cancel := context.WithTimeout(wp.ctx, 120*time.Second)
	defer cancel()

	ctx, span := jobTracer.Start(ctx, "job.execute",
		trace.WithAttributes(
			attribute.Int("worker.id", workerID),
			attribute.String("job.description", job.Description()),
			attribute.String("job.user_id", job.UserID()),
		),
	)
	defer span.End()

	start := time.Now()

	// Execute the job
	if err := job.Execute(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		jobTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "error")))
		jobDuration.Record(ctx, time.Since(start).Seconds())
		log.Printf("Worker %d: Error processing %s for user %s: %v",
			workerID, job.Description(), job.UserID(), err)
		return
	}

	jobTotal.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "success")))
	jobDuration.Record(ctx, time.Since(start).Seconds())
	log.Printf("Worker %d: Successfully completed %s for user %s",
		workerID, job.Description(), job.UserID())
}

// Submit adds a job to the queue for processing.
// Returns an error if the context is cancelled.
// Returns ErrQueueFull if the queue is full (job is dropped).
// Non-blocking: uses select to respect context cancellation.
func (wp *WorkerPool) Submit(job Job) error {
	select {
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	case wp.jobs <- job:
		return nil
	default:
		// Queue is full - could also block here, but we return error for visibility
		jobQueueDropped.Add(context.Background(), 1)
		log.Printf("Warning: Job queue full, dropping job for user %s", job.UserID())
		return fmt.Errorf("job queue full, dropping job for user %s", job.UserID())
	}
}

// SubmitBatch adds multiple jobs to the queue.
// Useful for batch processing scenarios (e.g., syncing all users).
func (wp *WorkerPool) SubmitBatch(jobs []Job) {
	submitted := 0
	for _, job := range jobs {
		if err := wp.Submit(job); err != nil {
			log.Printf("Failed to submit job for user %s: %v", job.UserID(), err)
			continue
		}
		submitted++
	}
	log.Printf("Submitted %d/%d jobs to worker pool", submitted, len(jobs))
}

// Shutdown gracefully stops the worker pool.
// It closes the job channel, waits for workers to finish, then cancels the context.
// Demonstrates: graceful shutdown, WaitGroups, context cancellation.
func (wp *WorkerPool) Shutdown() {
	log.Println("Worker pool: Initiating graceful shutdown")

	// Close the job channel to signal no more jobs will be added
	close(wp.jobs)

	// Wait for all workers to finish processing current jobs
	log.Println("Worker pool: Waiting for workers to finish...")
	wp.wg.Wait()

	// Cancel context to signal any long-running operations
	wp.cancel()

	log.Println("Worker pool: Shutdown complete")
}

// ShutdownWithTimeout shuts down the worker pool with a timeout.
// If workers don't finish within the timeout, it forces shutdown by cancelling context.
func (wp *WorkerPool) ShutdownWithTimeout(timeout time.Duration) {
	log.Printf("Worker pool: Initiating graceful shutdown with %v timeout", timeout)

	// Close the job channel
	close(wp.jobs)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Worker pool: All workers finished gracefully")
	case <-time.After(timeout):
		log.Println("Worker pool: Timeout reached, forcing shutdown")
		wp.cancel()
	}

	log.Println("Worker pool: Shutdown complete")
}

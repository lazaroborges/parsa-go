package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ScheduleTime represents a specific time of day when the scheduler should run.
type ScheduleTime struct {
	Hour   int
	Minute int
}

// String returns the time in HH:MM format.
func (st ScheduleTime) String() string {
	return fmt.Sprintf("%02d:%02d", st.Hour, st.Minute)
}

// ParseScheduleTime parses a time string in HH:MM format.
func ParseScheduleTime(s string) (ScheduleTime, error) {
	var hour, minute int
	_, err := fmt.Sscanf(s, "%d:%d", &hour, &minute)
	if err != nil {
		return ScheduleTime{}, fmt.Errorf("invalid time format (expected HH:MM): %w", err)
	}

	if hour < 0 || hour > 23 {
		return ScheduleTime{}, fmt.Errorf("invalid hour: %d (must be 0-23)", hour)
	}
	if minute < 0 || minute > 59 {
		return ScheduleTime{}, fmt.Errorf("invalid minute: %d (must be 0-59)", minute)
	}

	return ScheduleTime{Hour: hour, Minute: minute}, nil
}

// Scheduler manages periodic execution of jobs at specific times.
type Scheduler struct {
	workerPool    *WorkerPool
	scheduleTimes []ScheduleTime
	runOnStartup  bool
	jobProvider   func(context.Context) ([]Job, error)

	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	lastRunDate string
	mu          sync.RWMutex
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	ScheduleTimes []string
	WorkerCount   int
	JobDelay      time.Duration
	QueueSize     int
	RunOnStartup  bool
	JobProvider   func(context.Context) ([]Job, error)
}

// NewScheduler creates a new scheduler with the given configuration.
func NewScheduler(config SchedulerConfig) (*Scheduler, error) {
	scheduleTimes := make([]ScheduleTime, 0, len(config.ScheduleTimes))
	for _, timeStr := range config.ScheduleTimes {
		st, err := ParseScheduleTime(timeStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse schedule time %q: %w", timeStr, err)
		}
		scheduleTimes = append(scheduleTimes, st)
	}

	if len(scheduleTimes) == 0 {
		return nil, fmt.Errorf("at least one schedule time is required")
	}

	workerPool := NewWorkerPool(config.WorkerCount, config.JobDelay, config.QueueSize)
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Scheduler initialized with %d schedule times: %v", len(scheduleTimes), config.ScheduleTimes)
	log.Printf("Worker pool: %d workers, %v delay between jobs", config.WorkerCount, config.JobDelay)

	return &Scheduler{
		workerPool:    workerPool,
		scheduleTimes: scheduleTimes,
		runOnStartup:  config.RunOnStartup,
		jobProvider:   config.JobProvider,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start launches the scheduler and worker pool.
func (s *Scheduler) Start() {
	log.Println("Starting scheduler...")

	s.workerPool.Start()

	if s.runOnStartup {
		log.Println("Scheduler: Running initial job batch on startup")
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runJobs()
		}()
	}

	s.wg.Add(1)
	go s.scheduleLoop()

	log.Println("Scheduler started")
}

// scheduleLoop is the main scheduling loop.
func (s *Scheduler) scheduleLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Scheduler loop started, checking every minute")

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Scheduler loop: Context cancelled, shutting down")
			return

		case now := <-ticker.C:
			if s.shouldRun(now) {
				log.Printf("Scheduler: Triggered at %s", now.Format("15:04"))
				s.runJobs()
			}
		}
	}
}

// shouldRun checks if the current time matches any scheduled time.
func (s *Scheduler) shouldRun(now time.Time) bool {
	currentHour := now.Hour()
	currentMinute := now.Minute()
	currentKey := fmt.Sprintf("%s-%02d:%02d", now.Format("2006-01-02"), currentHour, currentMinute)

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastRunDate == currentKey {
		return false
	}

	for _, st := range s.scheduleTimes {
		if currentHour == st.Hour && currentMinute == st.Minute {
			s.lastRunDate = currentKey
			return true
		}
	}

	return false
}

// runJobs executes the job provider and submits jobs to the worker pool.
func (s *Scheduler) runJobs() {
	if s.jobProvider == nil {
		log.Println("Scheduler: No job provider configured")
		return
	}

	log.Println("Scheduler: Fetching jobs...")

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	jobs, err := s.jobProvider(ctx)
	if err != nil {
		log.Printf("Scheduler: Failed to fetch jobs: %v", err)
		return
	}

	if len(jobs) == 0 {
		log.Println("Scheduler: No jobs to process")
		return
	}

	log.Printf("Scheduler: Submitting %d jobs to worker pool", len(jobs))
	s.workerPool.SubmitBatch(jobs)
}

// Shutdown gracefully stops the scheduler and worker pool.
func (s *Scheduler) Shutdown(timeout time.Duration) {
	log.Println("Scheduler: Initiating graceful shutdown...")

	s.cancel()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Scheduler: Scheduler loop stopped gracefully")
	case <-time.After(timeout):
		log.Println("Scheduler: Timeout waiting for scheduler loop to stop")
	}

	s.workerPool.ShutdownWithTimeout(timeout)

	log.Println("Scheduler: Shutdown complete")
}

// TriggerNow manually triggers a job run immediately.
func (s *Scheduler) TriggerNow() {
	log.Println("Scheduler: Manual trigger")
	go s.runJobs()
}

// GetNextScheduledTime returns the next scheduled run time.
func (s *Scheduler) GetNextScheduledTime() time.Time {
	now := time.Now()

	for _, st := range s.scheduleTimes {
		scheduledTime := time.Date(now.Year(), now.Month(), now.Day(), st.Hour, st.Minute, 0, 0, now.Location())
		if scheduledTime.After(now) {
			return scheduledTime
		}
	}

	if len(s.scheduleTimes) > 0 {
		st := s.scheduleTimes[0]
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), st.Hour, st.Minute, 0, 0, now.Location())
	}

	return time.Time{}
}

// GetScheduleTimes returns the configured schedule times.
func (s *Scheduler) GetScheduleTimes() []ScheduleTime {
	return s.scheduleTimes
}

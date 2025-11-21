package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"parsa/internal/database"
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

// Scheduler manages periodic execution of sync jobs at specific times.
// It demonstrates Go's concurrency with Ticker, select, context, and channels.
type Scheduler struct {
	userRepo      *database.UserRepository
	pierreClient  PierreFinanceClient
	workerPool    *WorkerPool
	scheduleTimes []ScheduleTime
	runOnStartup  bool

	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	lastRunDate string // Track last run date (YYYY-MM-DD) to prevent multiple runs
	mu          sync.RWMutex
}

// SchedulerConfig holds configuration for the scheduler.
type SchedulerConfig struct {
	ScheduleTimes []string      // Schedule times in HH:MM format (e.g., ["05:00", "10:00", "14:00", "20:00"])
	WorkerCount   int           // Number of concurrent workers
	JobDelay      time.Duration // Delay between jobs (for rate limiting)
	QueueSize     int           // Job queue buffer size
	RunOnStartup  bool          // Run sync immediately on startup
}

// NewScheduler creates a new scheduler with the given configuration.
func NewScheduler(
	userRepo *database.UserRepository,
	pierreClient PierreFinanceClient,
	config SchedulerConfig,
) (*Scheduler, error) {
	// Parse schedule times
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

	// Create worker pool
	workerPool := NewWorkerPool(config.WorkerCount, config.JobDelay, config.QueueSize)

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	log.Printf("Scheduler initialized with %d schedule times: %v", len(scheduleTimes), config.ScheduleTimes)
	log.Printf("Worker pool: %d workers, %v delay between jobs", config.WorkerCount, config.JobDelay)

	return &Scheduler{
		userRepo:      userRepo,
		pierreClient:  pierreClient,
		workerPool:    workerPool,
		scheduleTimes: scheduleTimes,
		runOnStartup:  config.RunOnStartup,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start launches the scheduler and worker pool.
// It runs in a goroutine and checks every minute if it's time to run.
// Demonstrates: goroutines, Ticker, select with context, channels.
func (s *Scheduler) Start() {
	log.Println("Starting scheduler...")

	// Start worker pool
	s.workerPool.Start()

	// Run initial sync if configured
	if s.runOnStartup {
		log.Println("Scheduler: Running initial sync on startup")
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runSync()
		}()
	}

	// Start scheduler loop
	s.wg.Add(1)
	go s.scheduleLoop()

	log.Println("Scheduler started")
}

// scheduleLoop is the main scheduling loop.
// It uses a Ticker to check every minute if it's time to run.
// Demonstrates: Ticker, select with context, time-based scheduling.
func (s *Scheduler) scheduleLoop() {
	defer s.wg.Done()

	// Create ticker that fires every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Scheduler loop started, checking every minute")

	for {
		select {
		case <-s.ctx.Done():
			log.Println("Scheduler loop: Context cancelled, shutting down")
			return

		case now := <-ticker.C:
			// Check if current time matches any scheduled time
			if s.shouldRun(now) {
				log.Printf("Scheduler: Triggered at %s", now.Format("15:04"))
				s.runSync()
			}
		}
	}
}

// shouldRun checks if the current time matches any scheduled time.
// It also prevents running multiple times in the same minute by tracking last run.
func (s *Scheduler) shouldRun(now time.Time) bool {
	currentHour := now.Hour()
	currentMinute := now.Minute()
	currentKey := fmt.Sprintf("%s-%02d:%02d", now.Format("2006-01-02"), currentHour, currentMinute)

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if we already ran at this exact time today
	if s.lastRunDate == currentKey {
		return false
	}

	// Check if current time matches any scheduled time
	for _, st := range s.scheduleTimes {
		if currentHour == st.Hour && currentMinute == st.Minute {
			s.lastRunDate = currentKey
			return true
		}
	}

	return false
}

// runSync executes a sync run for all users.
// It fetches all users from the database and creates sync jobs for each.
func (s *Scheduler) runSync() {
	log.Println("Starting sync run for all users...")

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Minute)
	defer cancel()

	// Fetch all users
	users, err := s.userRepo.List(ctx)
	if err != nil {
		log.Printf("Scheduler: Failed to fetch users: %v", err)
		return
	}

	if len(users) == 0 {
		log.Println("Scheduler: No users to sync")
		return
	}

	log.Printf("Scheduler: Creating sync jobs for %d users", len(users))

	// Create sync jobs for each user
	jobs := make([]Job, 0, len(users))
	for _, user := range users {
		job := NewSyncJob(user, s.pierreClient)
		jobs = append(jobs, job)
	}

	// Submit jobs to worker pool
	s.workerPool.SubmitBatch(jobs)

	log.Printf("Scheduler: Submitted %d sync jobs to worker pool", len(jobs))
}

// Shutdown gracefully stops the scheduler and worker pool.
// It waits for the scheduler loop to finish and then shuts down the worker pool.
// Demonstrates: graceful shutdown, WaitGroups, context cancellation, timeouts.
func (s *Scheduler) Shutdown(timeout time.Duration) {
	log.Println("Scheduler: Initiating graceful shutdown...")

	// Cancel context to stop scheduler loop
	s.cancel()

	// Wait for scheduler loop to finish with timeout
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

	// Shutdown worker pool
	s.workerPool.ShutdownWithTimeout(timeout)

	log.Println("Scheduler: Shutdown complete")
}

// TriggerNow manually triggers a sync run immediately.
// Useful for testing or manual sync triggers via API endpoint.
func (s *Scheduler) TriggerNow() {
	log.Println("Scheduler: Manual sync triggered")
	go s.runSync()
}

// GetNextScheduledTime returns the next scheduled run time.
// Useful for monitoring and displaying to users.
func (s *Scheduler) GetNextScheduledTime() time.Time {
	now := time.Now()

	// Find the next scheduled time today or tomorrow
	for _, st := range s.scheduleTimes {
		scheduledTime := time.Date(now.Year(), now.Month(), now.Day(), st.Hour, st.Minute, 0, 0, now.Location())
		if scheduledTime.After(now) {
			return scheduledTime
		}
	}

	// No more scheduled times today, return first time tomorrow
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

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"parsa/internal/interfaces/scheduler"
	"parsa/internal/shared/config"
	"parsa/internal/shared/telemetry"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func run() error {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Initialize telemetry if enabled
	if cfg.Telemetry.Enabled {
		tp, err := telemetry.Init(context.Background(), telemetry.Config{
			ServiceName:  cfg.Telemetry.ServiceName,
			OTLPEndpoint: cfg.Telemetry.OTLPEndpoint,
		})
		if err != nil {
			return err
		}
		defer tp.Shutdown(context.Background())
		log.Printf("OpenTelemetry enabled (endpoint: %s)", cfg.Telemetry.OTLPEndpoint)
	}

	// Initialize dependencies
	deps, err := NewDependencies(cfg)
	if err != nil {
		return err
	}
	defer deps.Close()

	// Register DB pool metrics (no-op if telemetry is disabled)
	deps.DB.RegisterMetrics()

	// Setup routes and middleware
	handler := SetupRoutes(deps, cfg)

	// Initialize scheduler if enabled
	var sched *scheduler.Scheduler
	if cfg.Scheduler.Enabled {
		sched, err = setupScheduler(deps, cfg)
		if err != nil {
			return err
		}
		sched.Start()
		log.Println("Sync scheduler started (accounts + transactions)")
	}

	// Start servers
	scfg := NewServerConfigFromConfig(handler, cfg)
	srv, redirectSrv := StartServers(scfg)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	GracefulShutdown(srv, redirectSrv, sched, 30*time.Second)

	return nil
}

// setupScheduler initializes the background sync scheduler.
func setupScheduler(deps *Dependencies, cfg *config.Config) (*scheduler.Scheduler, error) {
	jobProvider := func(ctx context.Context) ([]scheduler.Job, error) {
		users, err := deps.UserRepo.ListUsersWithProviderKey(ctx)
		if err != nil {
			return nil, err
		}

		jobs := make([]scheduler.Job, 0, len(users))
		for _, user := range users {
			job := scheduler.NewUserSyncJob(user.ID, deps.AccountSyncService, deps.TransactionSyncService, deps.BillSyncService)
			jobs = append(jobs, job)
		}

		log.Printf("Job provider: Created %d sync jobs (%d users)", len(jobs), len(users))
		return jobs, nil
	}

	return scheduler.NewScheduler(scheduler.SchedulerConfig{
		ScheduleTimes: cfg.Scheduler.ScheduleTimes,
		WorkerCount:   cfg.Scheduler.WorkerCount,
		JobDelay:      cfg.Scheduler.JobDelay,
		QueueSize:     cfg.Scheduler.QueueSize,
		RunOnStartup:  cfg.Scheduler.RunOnStartup,
		JobProvider:   jobProvider,
	})
}

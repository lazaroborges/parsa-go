package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"parsa/internal/domain/bill"
	"parsa/internal/domain/transaction"
	"parsa/internal/infrastructure/crypto"
	"parsa/internal/infrastructure/postgres"
	"parsa/internal/shared/config"
)

const usage = `Parsa Admin CLI - Management commands for the Parsa API

Usage:
  admin <command> [options]

Commands:
  duplicate-check    Run duplicate transaction detection on existing transactions

Examples:
  # Check all transactions for a specific user
  admin duplicate-check --user-id=1

  # Check all transactions for multiple users
  admin duplicate-check --user-id=1,2,3

  # Check all transactions for all users
  admin duplicate-check --all

  # Run with custom worker count for higher concurrency
  admin duplicate-check --all --workers=8

  # Run with timeout
  admin duplicate-check --user-id=1 --timeout=5m
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "duplicate-check":
		runDuplicateCheck(os.Args[2:])
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		fmt.Print(usage)
		os.Exit(1)
	}
}

func runDuplicateCheck(args []string) {
	fs := flag.NewFlagSet("duplicate-check", flag.ExitOnError)

	userIDStr := fs.String("user-id", "", "User ID(s) to check (comma-separated for multiple)")
	allUsers := fs.Bool("all", false, "Check all users with transactions")
	workers := fs.Int("workers", transaction.DefaultWorkerCount, "Number of concurrent workers")
	timeoutStr := fs.String("timeout", "30m", "Timeout for the operation (e.g., 5m, 1h)")

	fs.Usage = func() {
		fmt.Println("Usage: admin duplicate-check [options]")
		fmt.Println("\nOptions:")
		fs.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  admin duplicate-check --user-id=1")
		fmt.Println("  admin duplicate-check --user-id=1,2,3")
		fmt.Println("  admin duplicate-check --all")
		fmt.Println("  admin duplicate-check --all --workers=8 --timeout=1h")
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *userIDStr == "" && !*allUsers {
		fmt.Println("Error: must specify --user-id or --all")
		fs.Usage()
		os.Exit(1)
	}

	// Parse timeout
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		log.Fatalf("Invalid timeout format: %v", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database
	db, err := postgres.New(cfg.Database.ConnectionString())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("Connected to database")

	// Initialize repositories
	transactionRepo := postgres.NewTransactionRepository(db)
	billRepo := postgres.NewBillRepository(db)

	// Initialize duplicate check service
	dupService := transaction.NewDuplicateCheckServiceWithWorkers(transactionRepo, *workers)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var userIDs []int64

	if *allUsers {
		// Need user repo to list all users
		encryptor, err := crypto.NewEncryptor(cfg.Encryption.Key)
		if err != nil {
			log.Fatalf("Failed to create encryptor: %v", err)
		}
		userRepo := postgres.NewUserRepository(db, encryptor)

		users, err := userRepo.ListUsersWithProviderKey(ctx)
		if err != nil {
			log.Fatalf("Failed to list users: %v", err)
		}

		for _, u := range users {
			userIDs = append(userIDs, u.ID)
		}
		log.Printf("Found %d users with provider keys", len(userIDs))
	} else {
		// Parse user IDs from comma-separated string
		parts := strings.Split(*userIDStr, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			id, err := strconv.ParseInt(p, 10, 64)
			if err != nil {
				log.Fatalf("Invalid user ID '%s': %v", p, err)
			}
			userIDs = append(userIDs, id)
		}
	}

	if len(userIDs) == 0 {
		log.Println("No users to process")
		return
	}

	log.Printf("Starting duplicate check for %d user(s) with %d workers", len(userIDs), *workers)
	startTime := time.Now()

	// Run duplicate check
	if len(userIDs) == 1 {
		// Single user - run directly
		result, err := dupService.CheckAllUserTransactions(ctx, userIDs[0])
		if err != nil {
			log.Fatalf("Duplicate check failed: %v", err)
		}
		printResult(userIDs[0], result)

		// Run bill duplicate check
		found, marked := checkBillDuplicates(ctx, userIDs[0], dupService, billRepo)
		printBillResult(userIDs[0], found, marked)
	} else {
		// Multiple users - run concurrently
		results := dupService.CheckAllUsersTransactions(ctx, userIDs)
		for uid, result := range results {
			printResult(uid, result)
		}

		// Run bill duplicate check for all users
		for _, uid := range userIDs {
			found, marked := checkBillDuplicates(ctx, uid, dupService, billRepo)
			printBillResult(uid, found, marked)
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("Duplicate check completed in %v", elapsed)
}

func printResult(userID int64, result *transaction.DuplicateCheckResult) {
	fmt.Printf("\n=== User %d (Transaction Duplicates) ===\n", userID)
	fmt.Printf("  Transactions checked: %d\n", result.TransactionsChecked)
	fmt.Printf("  Duplicates found:     %d\n", result.DuplicatesFound)
	fmt.Printf("  Duplicates marked:    %d\n", result.DuplicatesMarked)

	if len(result.Errors) > 0 {
		fmt.Printf("  Errors:               %d\n", len(result.Errors))
		for i, e := range result.Errors {
			if i >= 5 {
				fmt.Printf("    ... and %d more errors\n", len(result.Errors)-5)
				break
			}
			fmt.Printf("    - %s\n", e)
		}
	}
}

func checkBillDuplicates(ctx context.Context, userID int64, dupService *transaction.DuplicateCheckService, billRepo bill.Repository) (found int, marked int) {
	bills, err := billRepo.ListByUserID(ctx, userID, 1000, 0)
	if err != nil {
		log.Printf("Error fetching bills for user %d: %v", userID, err)
		return 0, 0
	}

	totalFound := 0
	totalMarked := 0

	for _, b := range bills {
		found, marked, err := dupService.CheckBillForDuplicates(ctx, b.AccountID, b.DueDate, b.TotalAmount, userID)
		if err != nil {
			log.Printf("Error checking bill duplicates for bill %s: %v", b.ID, err)
			continue
		}
		totalFound += found
		totalMarked += marked
	}

	return totalFound, totalMarked
}

func printBillResult(userID int64, found, marked int) {
	fmt.Printf("\n=== User %d (Bill Duplicates) ===\n", userID)
	fmt.Printf("  Duplicates found:  %d\n", found)
	fmt.Printf("  Duplicates marked: %d\n", marked)
}

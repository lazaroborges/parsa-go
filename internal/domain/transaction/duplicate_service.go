package transaction

import (
	"context"
	"log"
	"math"
	"strings"
	"sync"
	"time"
)

const (
	// DuplicateTimeDelta is the time window for finding potential duplicates (24 hours)
	DuplicateTimeDelta = 24 * time.Hour

	// DuplicateNote is the note added to transactions marked as potential duplicates
	DuplicateNote = "Esta transação não será considerada no cálculo de saldos e insights. Possíveis motivos incluem estornos, pagamentos e créditos da fatura do cartão de crédito, etc."

	// DefaultWorkerCount is the default number of concurrent workers for duplicate checking
	DefaultWorkerCount = 4
)

// DuplicateCheckResult contains the results of a duplicate check operation
type DuplicateCheckResult struct {
	TransactionsChecked int
	DuplicatesFound     int
	DuplicatesMarked    int
	Errors              []string
}

// duplicateCheckJob represents a single transaction to check for duplicates
type duplicateCheckJob struct {
	transaction *Transaction
	userID      int64
}

// duplicateCheckWorkerResult represents the result of processing a single job
type duplicateCheckWorkerResult struct {
	duplicatesFound  int
	duplicatesMarked int
	err              error
}

// DuplicateCheckService handles checking transactions for potential duplicates
type DuplicateCheckService struct {
	repo        Repository
	workerCount int
}

// NewDuplicateCheckService creates a new duplicate check service
func NewDuplicateCheckService(repo Repository) *DuplicateCheckService {
	return &DuplicateCheckService{
		repo:        repo,
		workerCount: DefaultWorkerCount,
	}
}

// NewDuplicateCheckServiceWithWorkers creates a new duplicate check service with custom worker count
func NewDuplicateCheckServiceWithWorkers(repo Repository, workerCount int) *DuplicateCheckService {
	if workerCount <= 0 {
		workerCount = DefaultWorkerCount
	}
	return &DuplicateCheckService{
		repo:        repo,
		workerCount: workerCount,
	}
}

// CheckBatchForDuplicates checks a batch of transactions for potential duplicates concurrently
// This is the main entry point for duplicate checking after batch operations
func (s *DuplicateCheckService) CheckBatchForDuplicates(ctx context.Context, transactions []*Transaction, userID int64) *DuplicateCheckResult {
	result := &DuplicateCheckResult{
		TransactionsChecked: len(transactions),
		Errors:              []string{},
	}

	if len(transactions) == 0 {
		return result
	}

	// Create channels for job distribution and result collection
	jobs := make(chan duplicateCheckJob, len(transactions))
	results := make(chan duplicateCheckWorkerResult, len(transactions))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go s.duplicateCheckWorker(ctx, jobs, results, &wg)
	}

	// Send jobs to workers
	for _, txn := range transactions {
		jobs <- duplicateCheckJob{
			transaction: txn,
			userID:      userID,
		}
	}
	close(jobs)

	// Wait for all workers to complete and close results channel
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	for workerResult := range results {
		if workerResult.err != nil {
			result.Errors = append(result.Errors, workerResult.err.Error())
		}
		result.DuplicatesFound += workerResult.duplicatesFound
		result.DuplicatesMarked += workerResult.duplicatesMarked
	}

	log.Printf("Duplicate check completed: checked=%d, found=%d, marked=%d, errors=%d",
		result.TransactionsChecked, result.DuplicatesFound, result.DuplicatesMarked, len(result.Errors))

	return result
}

// duplicateCheckWorker is a worker goroutine that processes duplicate check jobs
func (s *DuplicateCheckService) duplicateCheckWorker(
	ctx context.Context,
	jobs <-chan duplicateCheckJob,
	results chan<- duplicateCheckWorkerResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- duplicateCheckWorkerResult{err: ctx.Err()}
			return
		default:
			found, marked, err := s.checkTransactionForDuplicates(ctx, job.transaction, job.userID)
			results <- duplicateCheckWorkerResult{
				duplicatesFound:  found,
				duplicatesMarked: marked,
				err:              err,
			}
		}
	}
}

// checkTransactionForDuplicates checks a single transaction for duplicates and marks them
func (s *DuplicateCheckService) checkTransactionForDuplicates(
	ctx context.Context,
	txn *Transaction,
	userID int64,
) (found int, marked int, err error) {
	// Determine the opposite type
	oppositeType := "CREDIT"
	if txn.Type == "CREDIT" {
		oppositeType = "DEBIT"
	}

	// Calculate time bounds
	lowerBound := txn.TransactionDate.Add(-DuplicateTimeDelta)
	upperBound := txn.TransactionDate.Add(DuplicateTimeDelta)

	// Build search criteria
	criteria := DuplicateCriteria{
		ExcludeID:      txn.ID,
		OppositeType:   oppositeType,
		AbsoluteAmount: math.Abs(txn.Amount),
		DateLowerBound: lowerBound,
		DateUpperBound: upperBound,
		UserID:         userID,
	}

	// Find potential duplicates
	duplicates, err := s.repo.FindPotentialDuplicates(ctx, criteria)
	if err != nil {
		return 0, 0, err
	}

	found = len(duplicates)
	if found == 0 {
		return 0, 0, nil
	}

	// Mark duplicates as not considered
	for _, dup := range duplicates {
		if dup.Manipulated {
			continue // Skip manipulated transactions
		}

		// Check if note already contains the duplicate message
		if dup.Notes != nil {
			if strings.Contains(*dup.Notes, "Esta transação não será considerada") ||
				strings.Contains(*dup.Notes, "desconsiderada") {
				continue // Already marked
			}
		}

		// Update the duplicate transaction
		considered := false
		newNotes := DuplicateNote
		if dup.Notes != nil && *dup.Notes != "" {
			newNotes = *dup.Notes + " " + DuplicateNote
		}

		_, err := s.repo.Update(ctx, dup.ID, UpdateTransactionParams{
			Considered: &considered,
			Notes:      &newNotes,
		})
		if err != nil {
			log.Printf("Failed to mark transaction %s as duplicate: %v", dup.ID, err)
			continue
		}

		marked++
	}

	return found, marked, nil
}

// CheckTransactionForDuplicates checks a single transaction for duplicates
// This is useful for checking individual transactions during sync
func (s *DuplicateCheckService) CheckTransactionForDuplicates(
	ctx context.Context,
	txn *Transaction,
	userID int64,
) (duplicatesFound int, duplicatesMarked int, err error) {
	return s.checkTransactionForDuplicates(ctx, txn, userID)
}

// CheckBatchForDuplicatesConcurrent processes duplicate checking with a configurable concurrency level
// This allows fine-tuning performance based on system resources
func (s *DuplicateCheckService) CheckBatchForDuplicatesConcurrent(
	ctx context.Context,
	transactions []*Transaction,
	userID int64,
	concurrency int,
) *DuplicateCheckResult {
	if concurrency <= 0 {
		concurrency = s.workerCount
	}

	result := &DuplicateCheckResult{
		TransactionsChecked: len(transactions),
		Errors:              []string{},
	}

	if len(transactions) == 0 {
		return result
	}

	// Use semaphore pattern for bounded concurrency
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, txn := range transactions {
		wg.Add(1)
		go func(t *Transaction) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				result.Errors = append(result.Errors, ctx.Err().Error())
				mu.Unlock()
				return
			}

			found, marked, err := s.checkTransactionForDuplicates(ctx, t, userID)

			mu.Lock()
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
			}
			result.DuplicatesFound += found
			result.DuplicatesMarked += marked
			mu.Unlock()
		}(txn)
	}

	wg.Wait()

	log.Printf("Concurrent duplicate check completed: checked=%d, found=%d, marked=%d, errors=%d",
		result.TransactionsChecked, result.DuplicatesFound, result.DuplicatesMarked, len(result.Errors))

	return result
}

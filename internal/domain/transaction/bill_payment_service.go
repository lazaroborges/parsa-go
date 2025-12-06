package transaction

import (
	"context"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	"parsa/internal/domain/bill"
)

const (
	// BillPaymentTimeDelta is the time window for finding matching bills (120 hours = 5 days)
	BillPaymentTimeDelta = 120 * time.Hour

	// BillPaymentNote is the note added to transactions identified as credit card bill payments
	BillPaymentNote = "Reconhecida como pagamento de fatura de cartão de crédito e desconsiderada automaticamente."
)

// BillPaymentCheckResult contains the results of a bill payment check operation
type BillPaymentCheckResult struct {
	TransactionsChecked int
	BillPaymentsFound   int
	BillPaymentsMarked  int
	Errors              []string
}

// billPaymentCheckJob represents a single transaction to check for bill payment match
type billPaymentCheckJob struct {
	transaction *Transaction
}

// billPaymentCheckWorkerResult represents the result of processing a single job
type billPaymentCheckWorkerResult struct {
	found  bool
	marked bool
	err    error
}

// BillPaymentCheckService handles checking transactions for credit card bill payment matches
type BillPaymentCheckService struct {
	transactionRepo Repository
	billRepo        bill.Repository
	workerCount     int
}

// NewBillPaymentCheckService creates a new bill payment check service
func NewBillPaymentCheckService(transactionRepo Repository, billRepo bill.Repository) *BillPaymentCheckService {
	return &BillPaymentCheckService{
		transactionRepo: transactionRepo,
		billRepo:        billRepo,
		workerCount:     DefaultWorkerCount,
	}
}

// NewBillPaymentCheckServiceWithWorkers creates a new bill payment check service with custom worker count
func NewBillPaymentCheckServiceWithWorkers(transactionRepo Repository, billRepo bill.Repository, workerCount int) *BillPaymentCheckService {
	if workerCount <= 0 {
		workerCount = DefaultWorkerCount
	}
	return &BillPaymentCheckService{
		transactionRepo: transactionRepo,
		billRepo:        billRepo,
		workerCount:     workerCount,
	}
}

// CheckBatchForBillPayments checks a batch of transactions for credit card bill payment matches concurrently
func (s *BillPaymentCheckService) CheckBatchForBillPayments(ctx context.Context, transactions []*Transaction) *BillPaymentCheckResult {
	result := &BillPaymentCheckResult{
		TransactionsChecked: len(transactions),
		Errors:              []string{},
	}

	if len(transactions) == 0 {
		return result
	}

	// Create channels for job distribution and result collection
	jobs := make(chan billPaymentCheckJob, len(transactions))
	results := make(chan billPaymentCheckWorkerResult, len(transactions))

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < s.workerCount; i++ {
		wg.Add(1)
		go s.billPaymentCheckWorker(ctx, jobs, results, &wg)
	}

	// Send jobs to workers
	for _, txn := range transactions {
		jobs <- billPaymentCheckJob{
			transaction: txn,
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
		if workerResult.found {
			result.BillPaymentsFound++
		}
		if workerResult.marked {
			result.BillPaymentsMarked++
		}
	}

	log.Printf("Bill payment check completed: checked=%d, found=%d, marked=%d, errors=%d",
		result.TransactionsChecked, result.BillPaymentsFound, result.BillPaymentsMarked, len(result.Errors))

	return result
}

// billPaymentCheckWorker is a worker goroutine that processes bill payment check jobs
func (s *BillPaymentCheckService) billPaymentCheckWorker(
	ctx context.Context,
	jobs <-chan billPaymentCheckJob,
	results chan<- billPaymentCheckWorkerResult,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			results <- billPaymentCheckWorkerResult{err: ctx.Err()}
			return
		default:
			found, marked, err := s.checkTransactionForBillPayment(ctx, job.transaction)
			results <- billPaymentCheckWorkerResult{
				found:  found,
				marked: marked,
				err:    err,
			}
		}
	}
}

// checkTransactionForBillPayment checks a single transaction for bill payment match
func (s *BillPaymentCheckService) checkTransactionForBillPayment(
	ctx context.Context,
	txn *Transaction,
) (found bool, marked bool, err error) {
	// Skip if already not considered or manipulated
	if !txn.Considered || txn.Manipulated {
		return false, false, nil
	}

	// Calculate time bounds (±120 hours from transaction date)
	lowerBound := txn.TransactionDate.Add(-BillPaymentTimeDelta)
	upperBound := txn.TransactionDate.Add(BillPaymentTimeDelta)

	// Build search criteria
	criteria := bill.BillMatchCriteria{
		AccountID:      txn.AccountID,
		Amount:         math.Abs(txn.Amount),
		DateLowerBound: lowerBound,
		DateUpperBound: upperBound,
	}

	// Find matching bill
	matchingBill, err := s.billRepo.FindMatchingBill(ctx, criteria)
	if err != nil {
		return false, false, err
	}

	if matchingBill == nil {
		return false, false, nil
	}

	// Found a matching bill - mark transaction as not considered
	found = true

	// Check if note already contains the bill payment message
	if txn.Notes != nil {
		if strings.Contains(*txn.Notes, "pagamento de fatura") ||
			strings.Contains(*txn.Notes, "desconsiderada automaticamente") {
			return true, false, nil // Already marked
		}
	}

	// Update the transaction
	considered := false
	newNotes := BillPaymentNote
	if txn.Notes != nil && *txn.Notes != "" {
		newNotes = *txn.Notes + " " + BillPaymentNote
	}

	_, err = s.transactionRepo.Update(ctx, txn.ID, UpdateTransactionParams{
		Considered: &considered,
		Notes:      &newNotes,
	})
	if err != nil {
		log.Printf("Failed to mark transaction %s as bill payment: %v", txn.ID, err)
		return true, false, err
	}

	marked = true
	return found, marked, nil
}

// CheckTransactionForBillPayment checks a single transaction for bill payment match
// This is useful for checking individual transactions during sync
func (s *BillPaymentCheckService) CheckTransactionForBillPayment(
	ctx context.Context,
	txn *Transaction,
) (found bool, marked bool, err error) {
	return s.checkTransactionForBillPayment(ctx, txn)
}

// CheckBatchForBillPaymentsConcurrent processes bill payment checking with a configurable concurrency level
func (s *BillPaymentCheckService) CheckBatchForBillPaymentsConcurrent(
	ctx context.Context,
	transactions []*Transaction,
	concurrency int,
) *BillPaymentCheckResult {
	if concurrency <= 0 {
		concurrency = s.workerCount
	}

	result := &BillPaymentCheckResult{
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

			found, marked, err := s.checkTransactionForBillPayment(ctx, t)

			mu.Lock()
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
			}
			if found {
				result.BillPaymentsFound++
			}
			if marked {
				result.BillPaymentsMarked++
			}
			mu.Unlock()
		}(txn)
	}

	wg.Wait()

	log.Printf("Concurrent bill payment check completed: checked=%d, found=%d, marked=%d, errors=%d",
		result.TransactionsChecked, result.BillPaymentsFound, result.BillPaymentsMarked, len(result.Errors))

	return result
}

// CheckAllUserTransactions fetches and checks ALL existing transactions for a user
// for bill payment matches. Processes in batches to avoid memory issues.
func (s *BillPaymentCheckService) CheckAllUserTransactions(ctx context.Context, userID int64) (*BillPaymentCheckResult, error) {
	log.Printf("Starting full bill payment check for user %d", userID)

	totalResult := &BillPaymentCheckResult{
		Errors: []string{},
	}

	offset := 0
	batchNum := 0

	for {
		select {
		case <-ctx.Done():
			return totalResult, ctx.Err()
		default:
		}

		// Fetch a batch of transactions
		transactions, err := s.transactionRepo.ListByUserID(ctx, userID, DefaultBatchSize, offset)
		if err != nil {
			return totalResult, err
		}

		if len(transactions) == 0 {
			break // No more transactions
		}

		batchNum++
		log.Printf("Processing batch %d for bill payments: %d transactions (offset=%d)", batchNum, len(transactions), offset)

		// Process this batch concurrently
		batchResult := s.CheckBatchForBillPayments(ctx, transactions)

		// Aggregate results
		totalResult.TransactionsChecked += batchResult.TransactionsChecked
		totalResult.BillPaymentsFound += batchResult.BillPaymentsFound
		totalResult.BillPaymentsMarked += batchResult.BillPaymentsMarked
		totalResult.Errors = append(totalResult.Errors, batchResult.Errors...)

		// Move to next batch
		offset += len(transactions)

		// If we got fewer than batch size, we've reached the end
		if len(transactions) < DefaultBatchSize {
			break
		}
	}

	log.Printf("Full bill payment check completed for user %d: checked=%d, found=%d, marked=%d, errors=%d",
		userID, totalResult.TransactionsChecked, totalResult.BillPaymentsFound, totalResult.BillPaymentsMarked, len(totalResult.Errors))

	return totalResult, nil
}

// CheckAllUsersTransactions runs bill payment check for all provided user IDs concurrently
// Returns a map of userID -> result
func (s *BillPaymentCheckService) CheckAllUsersTransactions(ctx context.Context, userIDs []int64) map[int64]*BillPaymentCheckResult {
	results := make(map[int64]*BillPaymentCheckResult)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Use semaphore to limit concurrent user processing
	sem := make(chan struct{}, s.workerCount)

	for _, userID := range userIDs {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				results[uid] = &BillPaymentCheckResult{
					Errors: []string{ctx.Err().Error()},
				}
				mu.Unlock()
				return
			}

			result, err := s.CheckAllUserTransactions(ctx, uid)
			if err != nil {
				result = &BillPaymentCheckResult{
					Errors: []string{err.Error()},
				}
			}

			mu.Lock()
			results[uid] = result
			mu.Unlock()
		}(userID)
	}

	wg.Wait()
	return results
}

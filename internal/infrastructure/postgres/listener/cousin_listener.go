package listener

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"

	"github.com/lib/pq"

	"parsa/internal/domain/cousinrule"
)

const (
	channelName       = "cousin_assigned"
	reconnectInterval = 5 * time.Second
)

// CousinNotification represents the payload from PostgreSQL NOTIFY
type CousinNotification struct {
	TransactionID string `json:"transaction_id"`
	CousinID      int64  `json:"cousin_id"`
	TxType        string `json:"type"`
	AccountID     string `json:"account_id"`
}

// CousinListener listens for PostgreSQL notifications when transactions get cousins assigned
type CousinListener struct {
	connStr        string
	cousinRuleRepo cousinrule.Repository
	db             *sql.DB
	shutdownCh     chan struct{}
	done           chan struct{}
}

// NewCousinListener creates a new listener for cousin assignment notifications
func NewCousinListener(connStr string, cousinRuleRepo cousinrule.Repository, db *sql.DB) *CousinListener {
	return &CousinListener{
		connStr:        connStr,
		cousinRuleRepo: cousinRuleRepo,
		db:             db,
		shutdownCh:     make(chan struct{}),
		done:           make(chan struct{}),
	}
}

// Start begins listening for notifications in a background goroutine
func (l *CousinListener) Start(ctx context.Context) {
	go l.listen(ctx)
	log.Println("Cousin notification listener started")
}

// Stop gracefully shuts down the listener
func (l *CousinListener) Stop() {
	close(l.shutdownCh)
	<-l.done
	log.Println("Cousin notification listener stopped")
}

func (l *CousinListener) listen(ctx context.Context) {
	defer close(l.done)

	for {
		select {
		case <-l.shutdownCh:
			return
		case <-ctx.Done():
			return
		default:
			l.connectAndListen(ctx)
		}

		// Wait before reconnecting
		select {
		case <-l.shutdownCh:
			return
		case <-ctx.Done():
			return
		case <-time.After(reconnectInterval):
			log.Println("Reconnecting to PostgreSQL for notifications...")
		}
	}
}

func (l *CousinListener) connectAndListen(ctx context.Context) {
	// Create a dedicated listener connection
	listener := pq.NewListener(l.connStr, 10*time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("Listener error: %v", err)
		}
		switch ev {
		case pq.ListenerEventConnected:
			log.Println("Connected to PostgreSQL notification channel")
		case pq.ListenerEventDisconnected:
			log.Println("Disconnected from PostgreSQL notification channel")
		case pq.ListenerEventReconnected:
			log.Println("Reconnected to PostgreSQL notification channel")
		case pq.ListenerEventConnectionAttemptFailed:
			log.Printf("Connection attempt failed: %v", err)
		}
	})

	defer listener.Close()

	err := listener.Listen(channelName)
	if err != nil {
		log.Printf("Failed to listen on channel %s: %v", channelName, err)
		return
	}

	log.Printf("Listening on channel: %s", channelName)

	for {
		select {
		case <-l.shutdownCh:
			return
		case <-ctx.Done():
			return
		case notification := <-listener.Notify:
			if notification == nil {
				// Connection lost, break to reconnect
				return
			}
			l.handleNotification(ctx, notification)
		case <-time.After(90 * time.Second):
			// Ping to keep connection alive
			go func() {
				if err := listener.Ping(); err != nil {
					log.Printf("Listener ping failed: %v", err)
				}
			}()
		}
	}
}

func (l *CousinListener) handleNotification(ctx context.Context, notification *pq.Notification) {
	log.Printf("Received notification on channel %s", notification.Channel)

	var payload CousinNotification
	if err := json.Unmarshal([]byte(notification.Extra), &payload); err != nil {
		log.Printf("Failed to parse notification payload: %v", err)
		return
	}

	// Use background context since parent ctx may be cancelled during shutdown
	go l.applyCousinRule(context.Background(), payload)
}

func (l *CousinListener) applyCousinRule(ctx context.Context, payload CousinNotification) {
	// Get user_id from account
	var userID int64
	err := l.db.QueryRowContext(ctx,
		`SELECT user_id FROM accounts WHERE id = $1`,
		payload.AccountID,
	).Scan(&userID)

	if err != nil {
		log.Printf("Failed to get user_id for account %s: %v", payload.AccountID, err)
		return
	}

	// Find matching rule
	rule, err := l.findMatchingRule(ctx, userID, payload.CousinID, payload.TxType)
	if err != nil {
		log.Printf("Failed to find rule for cousin %d: %v", payload.CousinID, err)
		return
	}

	if rule == nil {
		log.Printf("No rule found for cousin %d, user %d, type %s", payload.CousinID, userID, payload.TxType)
		return
	}

	// Apply the rule to the transaction
	err = l.applyRuleToTransaction(ctx, payload.TransactionID, rule)
	if err != nil {
		log.Printf("Failed to apply rule to transaction %s: %v", payload.TransactionID, err)
		return
	}

	log.Printf("Successfully applied cousin rule to transaction")
}

func (l *CousinListener) findMatchingRule(ctx context.Context, userID, cousinID int64, txType string) (*cousinrule.CousinRule, error) {
	// Try type-specific rule first
	rule, err := l.cousinRuleRepo.GetByCousinAndType(ctx, userID, cousinID, &txType)
	if err != nil {
		return nil, err
	}
	if rule != nil {
		// Load tags
		tags, err := l.cousinRuleRepo.GetRuleTags(ctx, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Tags = tags
		return rule, nil
	}

	// Fall back to type-agnostic rule
	rule, err = l.cousinRuleRepo.GetByCousinAndType(ctx, userID, cousinID, nil)
	if err != nil {
		return nil, err
	}
	if rule != nil {
		tags, err := l.cousinRuleRepo.GetRuleTags(ctx, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.Tags = tags
	}

	return rule, nil
}

func (l *CousinListener) applyRuleToTransaction(ctx context.Context, transactionID string, rule *cousinrule.CousinRule) error {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Build dynamic UPDATE based on rule fields
	query := `
		UPDATE transactions
		SET manipulated = true,
		    updated_at = CURRENT_TIMESTAMP
	`
	args := []any{}
	argIndex := 1

	if rule.Category != nil {
		query += `, original_category = COALESCE(original_category, category)`
		query += `, category = $` + itoa(argIndex)
		args = append(args, *rule.Category)
		argIndex++
	}

	if rule.Description != nil {
		query += `, original_description = COALESCE(original_description, description)`
		query += `, description = $` + itoa(argIndex)
		args = append(args, *rule.Description)
		argIndex++
	}

	if rule.Notes != nil {
		query += `, notes = $` + itoa(argIndex)
		args = append(args, *rule.Notes)
		argIndex++
	}

	if rule.Considered != nil {
		query += `, considered = $` + itoa(argIndex)
		args = append(args, *rule.Considered)
		argIndex++
	}

	query += ` WHERE id = $` + itoa(argIndex)
	args = append(args, transactionID)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	// Apply tags if any
	if len(rule.Tags) > 0 {
		for _, tagID := range rule.Tags {
			_, err = tx.ExecContext(ctx,
				`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				transactionID, tagID,
			)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

// Simple int to string without importing strconv
func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return itoa(i/10) + string(rune('0'+i%10))
}

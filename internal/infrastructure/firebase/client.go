package firebase

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

const fcmBatchLimit = 500

// TokenDeactivator is called to mark an invalid FCM token as inactive.
// Provided by the caller (e.g. service layer) to avoid coupling to the repository.
type TokenDeactivator func(ctx context.Context, token string) error

// Client implements notification.Messenger using Firebase Cloud Messaging
type Client struct {
	msgClient   *messaging.Client
	deactivator TokenDeactivator
}

// NewClient initializes a Firebase app and returns an FCM client.
// deactivator is called when an invalid/unregistered token is detected; may be nil.
func NewClient(ctx context.Context, credentialsFile string, deactivator TokenDeactivator) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase messaging client: %w", err)
	}

	return &Client{msgClient: msgClient, deactivator: deactivator}, nil
}

// Send sends a push notification to a single device token
func (c *Client) Send(ctx context.Context, token string, title, body string, data map[string]string) error {
	msg := &messaging.Message{
		Token: token,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
	}

	_, err := c.msgClient.Send(ctx, msg)
	if err != nil {
		if messaging.IsUnregistered(err) || messaging.IsInvalidArgument(err) {
			log.Printf("Invalid FCM token (deactivating): %s", token)
			c.deactivateToken(ctx, token)
			return fmt.Errorf("invalid token: %w", err)
		}
		return fmt.Errorf("failed to send FCM message: %w", err)
	}

	return nil
}

// SendMulticast sends a push notification to multiple device tokens.
// Automatically batches into chunks of 500 (Firebase API limit).
func (c *Client) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}

	var totalSuccess, totalFailure int
	for _, batch := range chunkTokens(tokens, fcmBatchLimit) {
		msg := &messaging.MulticastMessage{
			Tokens: batch,
			Notification: &messaging.Notification{
				Title: title,
				Body:  body,
			},
			Data: data,
		}

		resp, err := c.msgClient.SendEachForMulticast(ctx, msg)
		if err != nil {
			return fmt.Errorf("failed to send FCM multicast: %w", err)
		}

		totalSuccess += resp.SuccessCount
		totalFailure += resp.FailureCount
		if resp.FailureCount > 0 {
			c.handleMulticastFailures(ctx, batch, resp)
		}
	}

	log.Printf("FCM multicast: %d success, %d failure", totalSuccess, totalFailure)
	return nil
}

// SendDataOnly sends a data-only message (no OS notification) to multiple tokens.
// Used for silent triggers like in-app reload — handled by onMessage in foreground only.
// Automatically batches into chunks of 500 (Firebase API limit).
func (c *Client) SendDataOnly(ctx context.Context, tokens []string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}

	var totalSuccess, totalFailure int
	for _, batch := range chunkTokens(tokens, fcmBatchLimit) {
		msg := &messaging.MulticastMessage{
			Tokens: batch,
			Data:   data,
		}

		resp, err := c.msgClient.SendEachForMulticast(ctx, msg)
		if err != nil {
			return fmt.Errorf("failed to send FCM data-only multicast: %w", err)
		}

		totalSuccess += resp.SuccessCount
		totalFailure += resp.FailureCount
		if resp.FailureCount > 0 {
			c.handleMulticastFailures(ctx, batch, resp)
		}
	}

	log.Printf("FCM data-only multicast: %d success, %d failure", totalSuccess, totalFailure)
	return nil
}

func (c *Client) handleMulticastFailures(ctx context.Context, tokens []string, resp *messaging.BatchResponse) {
	for i, sendResp := range resp.Responses {
		if sendResp.Error == nil {
			continue
		}
		if messaging.IsUnregistered(sendResp.Error) || messaging.IsInvalidArgument(sendResp.Error) {
			log.Printf("Invalid FCM token at index %d (deactivating token=%s): %v", i, tokens[i], sendResp.Error)
			c.deactivateToken(ctx, tokens[i])
		} else {
			log.Printf("FCM send error at index %d: %v", i, sendResp.Error)
		}
	}
}

func (c *Client) deactivateToken(ctx context.Context, token string) {
	if c.deactivator == nil {
		return
	}
	if err := c.deactivator(ctx, token); err != nil {
		log.Printf("Failed to deactivate FCM token %s: %v", token, err)
	}
}

func chunkTokens(tokens []string, size int) [][]string {
	var chunks [][]string
	for i := 0; i < len(tokens); i += size {
		end := i + size
		if end > len(tokens) {
			end = len(tokens)
		}
		chunks = append(chunks, tokens[i:end])
	}
	return chunks
}

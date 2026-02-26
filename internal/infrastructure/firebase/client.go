package firebase

import (
	"context"
	"fmt"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// Client implements notification.Messenger using Firebase Cloud Messaging
type Client struct {
	msgClient *messaging.Client
}

// NewClient initializes a Firebase app and returns an FCM client
func NewClient(ctx context.Context, credentialsFile string) (*Client, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase app: %w", err)
	}

	msgClient, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize firebase messaging client: %w", err)
	}

	return &Client{msgClient: msgClient}, nil
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
			log.Printf("Invalid FCM token (will be deactivated): %s", token)
			return fmt.Errorf("invalid token: %w", err)
		}
		return fmt.Errorf("failed to send FCM message: %w", err)
	}

	return nil
}

// SendMulticast sends a push notification to multiple device tokens
func (c *Client) SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if len(tokens) == 0 {
		return nil
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
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

	if resp.FailureCount > 0 {
		for i, sendResp := range resp.Responses {
			if sendResp.Error != nil {
				if messaging.IsUnregistered(sendResp.Error) || messaging.IsInvalidArgument(sendResp.Error) {
					log.Printf("Invalid FCM token at index %d (token=%s): %v", i, tokens[i], sendResp.Error)
				} else {
					log.Printf("FCM send error at index %d: %v", i, sendResp.Error)
				}
			}
		}
	}

	log.Printf("FCM multicast: %d success, %d failure", resp.SuccessCount, resp.FailureCount)
	return nil
}

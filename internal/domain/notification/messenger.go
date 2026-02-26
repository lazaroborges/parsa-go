package notification

import "context"

// Messenger defines the interface for sending push notifications.
// Implemented by the Firebase FCM client in the infrastructure layer.
type Messenger interface {
	Send(ctx context.Context, token string, title, body string, data map[string]string) error
	SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}

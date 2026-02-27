package messages

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed notifications.json
var notificationsJSON []byte

type MessageText struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type Messages struct {
	SyncComplete       MessageText `json:"sync_complete"`
	ProviderKeyCleared MessageText `json:"provider_key_cleared"`
}

var (
	loaded   Messages
	loadOnce sync.Once
	loadErr  error
)

// Load parses the embedded notifications JSON and caches the result.
// Safe to call from multiple goroutines.
func Load() (*Messages, error) {
	loadOnce.Do(func() {
		if err := json.Unmarshal(notificationsJSON, &loaded); err != nil {
			loadErr = err
		}
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return &loaded, nil
}

package messages

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

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

// Load reads the notifications JSON file and caches the result.
// Safe to call from multiple goroutines.
func Load(path string) (*Messages, error) {
	loadOnce.Do(func() {
		data, err := os.ReadFile(path)
		if err != nil {
			loadErr = fmt.Errorf("failed to read messages file: %w", err)
			return
		}
		if err := json.Unmarshal(data, &loaded); err != nil {
			loadErr = fmt.Errorf("failed to parse messages file: %w", err)
		}
	})
	if loadErr != nil {
		return nil, loadErr
	}
	return &loaded, nil
}

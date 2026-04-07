package konnectedkhbridge

import (
	"encoding/json"
	"os"
	"sync"
)

type PersistentState struct {
	mu           sync.Mutex
	filepath     string
	SecurityMode int `json:"security_mode"`
}

func NewPersistentState(path string) *PersistentState {
	return &PersistentState{
		filepath:     path,
		SecurityMode: 3, // Default to Disarmed
	}
}

func (p *PersistentState) UpdateAndSave(mode int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.SecurityMode = mode

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.filepath, data, 0644)
}

func (p *PersistentState) Load() error {
	data, err := os.ReadFile(p.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return p.UpdateAndSave(0)
		}
		return err
	}
	return json.Unmarshal(data, p)
}

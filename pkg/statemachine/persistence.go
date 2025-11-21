package statemachine

import (
	"encoding/json"
	"time"
)

// Snapshot 状态快照
type Snapshot struct {
	State     State     `json:"state"`
	Timestamp time.Time `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// History 状态历史记录
type History struct {
	From      State     `json:"from"`
	To        State     `json:"to"`
	Event     Event     `json:"event"`
	Timestamp time.Time `json:"timestamp"`
}

// PersistentFSM 支持持久化的状态机
type PersistentFSM struct {
	*FSM
	history []History
}

// NewPersistentFSM 创建支持持久化的状态机
func NewPersistentFSM(initial State) *PersistentFSM {
	return &PersistentFSM{
		FSM:     NewFSM(initial),
		history: make([]History, 0),
	}
}

// CreateSnapshot 创建状态快照
func (p *PersistentFSM) CreateSnapshot(metadata map[string]interface{}) *Snapshot {
	return &Snapshot{
		State:     p.Current(),
		Timestamp: time.Now(),
		Metadata:  metadata,
	}
}

// RestoreSnapshot 恢复状态快照
func (p *PersistentFSM) RestoreSnapshot(snapshot *Snapshot) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = snapshot.State
	return nil
}

// GetHistory 获取状态历史
func (p *PersistentFSM) GetHistory() []History {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]History{}, p.history...)
}

// ClearHistory 清空历史记录
func (p *PersistentFSM) ClearHistory() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.history = make([]History, 0)
}

// MarshalJSON 序列化状态机
func (p *PersistentFSM) MarshalJSON() ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data := map[string]interface{}{
		"current": p.current,
		"initial": p.initial,
		"history": p.history,
	}
	return json.Marshal(data)
}

// UnmarshalJSON 反序列化状态机
func (p *PersistentFSM) UnmarshalJSON(data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}

	if current, ok := obj["current"].(string); ok {
		p.current = State(current)
	}
	if initial, ok := obj["initial"].(string); ok {
		p.initial = State(initial)
	}

	return nil
}

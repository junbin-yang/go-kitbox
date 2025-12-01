package statemachine

import (
	"encoding/json"
	"testing"
)

func TestNewPersistentFSM(t *testing.T) {
	fsm := NewPersistentFSM("idle")
	if fsm == nil {
		t.Fatal("NewPersistentFSM returned nil")
	}
	if fsm.Current() != "idle" {
		t.Errorf("Expected initial state 'idle', got '%s'", fsm.Current())
	}
}

func TestCreateSnapshot(t *testing.T) {
	fsm := NewPersistentFSM("idle")
	metadata := map[string]interface{}{
		"version": "1.0",
		"user":    "test",
	}

	snapshot := fsm.CreateSnapshot(metadata)
	if snapshot == nil {
		t.Fatal("CreateSnapshot returned nil")
	}
	if snapshot.State != "idle" {
		t.Errorf("Expected snapshot state 'idle', got '%s'", snapshot.State)
	}
	if snapshot.Metadata["version"] != "1.0" {
		t.Error("Snapshot metadata not preserved")
	}
}

func TestRestoreSnapshot(t *testing.T) {
	fsm := NewPersistentFSM("idle")
	_ = fsm.AddTransition("idle", "running", "start")

	// Create snapshot in running state
	snapshot := &Snapshot{
		State: "running",
	}

	err := fsm.RestoreSnapshot(snapshot)
	if err != nil {
		t.Errorf("RestoreSnapshot failed: %v", err)
	}

	if fsm.Current() != "running" {
		t.Errorf("Expected state 'running' after restore, got '%s'", fsm.Current())
	}
}

func TestGetHistory(t *testing.T) {
	fsm := NewPersistentFSM("idle")

	history := fsm.GetHistory()
	if history == nil {
		t.Error("GetHistory returned nil")
	}
	if len(history) != 0 {
		t.Errorf("Expected empty history, got %d entries", len(history))
	}
}

func TestClearHistory(t *testing.T) {
	fsm := NewPersistentFSM("idle")

	fsm.ClearHistory()

	history := fsm.GetHistory()
	if len(history) != 0 {
		t.Errorf("Expected empty history after clear, got %d entries", len(history))
	}
}

func TestMarshalJSON(t *testing.T) {
	fsm := NewPersistentFSM("idle")
	_ = fsm.AddTransition("idle", "running", "start")

	data, err := fsm.MarshalJSON()
	if err != nil {
		t.Errorf("MarshalJSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("MarshalJSON returned empty data")
	}

	// Verify it's valid JSON
	var obj map[string]interface{}
	err = json.Unmarshal(data, &obj)
	if err != nil {
		t.Errorf("MarshalJSON produced invalid JSON: %v", err)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	fsm := NewPersistentFSM("idle")

	jsonData := []byte(`{"current":"running","initial":"idle","history":[]}`)
	err := fsm.UnmarshalJSON(jsonData)
	if err != nil {
		t.Errorf("UnmarshalJSON failed: %v", err)
	}

	if fsm.Current() != "running" {
		t.Errorf("Expected state 'running' after unmarshal, got '%s'", fsm.Current())
	}
}

func TestPersistentFSMRoundTrip(t *testing.T) {
	fsm1 := NewPersistentFSM("idle")
	_ = fsm1.AddTransition("idle", "running", "start")

	// Marshal
	data, err := fsm1.MarshalJSON()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal into new FSM
	fsm2 := NewPersistentFSM("idle")
	err = fsm2.UnmarshalJSON(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if fsm1.Current() != fsm2.Current() {
		t.Errorf("State mismatch after round trip: %s != %s", fsm1.Current(), fsm2.Current())
	}
}

package main

import (
	"sync"
	"time"
)

type AppState int

const (
	StateConnected AppState = iota
	StateReconnecting
	StatePaused
)

func (s AppState) String() string {
	m := T()
	switch s {
	case StateConnected:
		return m.StatusConnected
	case StateReconnecting:
		return m.StatusReconnecting
	case StatePaused:
		return m.StatusPaused
	default:
		return "Unknown"
	}
}

type StateManager struct {
	mu           sync.RWMutex
	state        AppState
	pauseUntil   time.Time
	retryCount   int
	lastError    string
	stateChanged chan struct{}
}

func newStateManager() *StateManager {
	return &StateManager{
		state:        StatePaused, // start paused until first check
		stateChanged: make(chan struct{}, 10),
	}
}

func (sm *StateManager) GetState() (AppState, time.Time, int) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state, sm.pauseUntil, sm.retryCount
}

func (sm *StateManager) SetConnected() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if sm.state != StateConnected || sm.retryCount != 0 {
		sm.state = StateConnected
		sm.pauseUntil = time.Time{}
		sm.retryCount = 0
		sm.lastError = ""
		sm.notify()
	}
}

func (sm *StateManager) SetReconnecting(attempt int, errMsg string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = StateReconnecting
	sm.retryCount = attempt
	sm.lastError = errMsg
	sm.notify()
}

func (sm *StateManager) SetPaused(until time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = StatePaused
	sm.pauseUntil = until
	sm.retryCount = 0
	sm.notify()
}

func (sm *StateManager) IsPaused() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if sm.state != StatePaused {
		return false
	}
	if !sm.pauseUntil.IsZero() && time.Now().After(sm.pauseUntil) {
		return false
	}
	return true
}

func (sm *StateManager) notify() {
	select {
	case sm.stateChanged <- struct{}{}:
	default:
	}
}

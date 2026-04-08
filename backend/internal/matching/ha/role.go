package ha

import (
	"sync"
)

// Role represents the HA role of a matching service instance.
type Role int

const (
	RolePrimary Role = iota
	RoleStandby
)

func (r Role) String() string {
	switch r {
	case RolePrimary:
		return "PRIMARY"
	case RoleStandby:
		return "STANDBY"
	default:
		return "UNKNOWN"
	}
}

// RoleManager tracks the current role and notifies listeners on transitions.
type RoleManager struct {
	mu        sync.RWMutex
	role      Role
	listeners []func(old, new Role)
}

func NewRoleManager(initial Role) *RoleManager {
	return &RoleManager{role: initial}
}

// Current returns the current role.
func (rm *RoleManager) Current() Role {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return rm.role
}

// IsPrimary returns true if the current role is PRIMARY.
func (rm *RoleManager) IsPrimary() bool {
	return rm.Current() == RolePrimary
}

// Transition changes the role and fires listeners if it changed.
func (rm *RoleManager) Transition(newRole Role) {
	rm.mu.Lock()
	old := rm.role
	if old == newRole {
		rm.mu.Unlock()
		return
	}
	rm.role = newRole
	listeners := make([]func(old, new Role), len(rm.listeners))
	copy(listeners, rm.listeners)
	rm.mu.Unlock()

	for _, fn := range listeners {
		fn(old, newRole)
	}
}

// OnTransition registers a callback for role changes.
func (rm *RoleManager) OnTransition(fn func(old, new Role)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.listeners = append(rm.listeners, fn)
}

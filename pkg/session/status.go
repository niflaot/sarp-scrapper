// Package session defines the authentication readiness contract shared by
// transport and browser infrastructure.
package session

type State string

const (
	StateAvailable State = "available"
	StatePending   State = "pending"
)

type Provider interface {
	Status() State
}

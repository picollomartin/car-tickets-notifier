package models

import (
	"strings"
	"time"
)

// District ...
type District interface {
	Name() string
	GetTickets(plateNumber string) ([]Ticket, error)
}

// Ticket ...
type Ticket struct {
	Description   string
	AmountInCents int64
	TicketNumber  string
	Date          time.Time
	DueDate       time.Time
}

// DescriptionEscaped ...
func (t *Ticket) DescriptionEscaped() string {
	return strings.ReplaceAll(t.Description, "*", "\\*")
}

// NotificationChannel ...
type NotificationChannel interface {
	Send(notification string) error
}

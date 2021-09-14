package models

import "time"

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

// NotificationChannel ...
type NotificationChannel interface {
	Send(notification string) error
}

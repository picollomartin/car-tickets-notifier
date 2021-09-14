package notifier

import (
	"fmt"
	"strings"
	"time"

	"github.com/leekchan/accounting"

	"car-tickets-notifier/internal/models"
	"github.com/google/martian/log"
)

// Notify ...
func Notify(notificationChannel models.NotificationChannel, plateNumber string, districts []models.District) {
	var msgs []string
	for _, district := range districts {
		tickets, err := district.GetTickets(plateNumber)
		if err != nil {
			log.Errorf("fail getting tickets from district %s: %s", district.Name(), err.Error())
		}
		msgs = append(msgs, formatTicketMessage(district, plateNumber, tickets))
	}
	reportMsg := strings.Join(msgs, "\n")
	if err := notificationChannel.Send(reportMsg); err != nil {
		log.Errorf("fail sending report of tickets: %s", err.Error())
	}
}

func formatTicketMessage(district models.District, plateNumber string, tickets []models.Ticket) string {
	districtTitle := fmt.Sprintf("Consulta de infracciones en %s (%s):", district.Name(), plateNumber)
	if len(tickets) == 0 {
		return fmt.Sprintf("%s 🎉 sin multas", districtTitle)
	}

	totalTicketsInCents := int64(0)
	for _, ticket := range tickets {
		totalTicketsInCents += ticket.AmountInCents
	}
	reportTitle := fmt.Sprintf("%s ❌ se encontraron *%d* multas por un total de *%s* 💰", districtTitle, len(tickets), formatMoney(totalTicketsInCents))
	var ticketsStrings []string

	for _, ticket := range tickets {
		ticketsStrings = append(ticketsStrings, fmt.Sprintf("• 💰 *%s* 🆔 %s 📕 %s 📅 %s ⏰ %s",
			formatMoney(ticket.AmountInCents), ticket.TicketNumber, ticket.Description, formatDate(ticket.Date), formatDate(ticket.DueDate)))
	}
	return fmt.Sprintf("%s\n%s", reportTitle, strings.Join(ticketsStrings, "\n"))
}

func formatMoney(cents int64) string {
	return fmt.Sprintf("$%s", accounting.FormatNumberFloat64(float64(cents)/100, 2, ".", ","))
}

func formatDate(date time.Time) string {
	return date.Format("2006-01-02")
}

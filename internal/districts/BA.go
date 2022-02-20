package districts

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"car-tickets-notifier/internal/models"
	"crypto/tls"
)

// BA ...
type BA struct {
	baseURL       string
	captchaSolver CaptchaSolver
}

// NewBA ...
func NewBA(baseURL string, captchaSolver CaptchaSolver) *BA {
	return &BA{
		captchaSolver: captchaSolver,
		baseURL:       baseURL,
	}
}

// Name ...
func (d *BA) Name() string {
	return "Buenos Aires"
}

// GetTickets ...
func (d *BA) GetTickets(plateNumber string) ([]models.Ticket, error) {
	captchaCode, err := d.captchaSolver.GetCaptchaCode()
	if err != nil {
		return nil, fmt.Errorf("fail getting captcha code: %w", err)
	}

	rawTickets, err := d.getBARawTickets(plateNumber, captchaCode)
	if err != nil {
		return nil, fmt.Errorf("fail getting tickets: %w", err)
	}
	return parseBARawTickets(rawTickets)

}

func parseBARawTickets(ticketResponse map[string]interface{}) ([]models.Ticket, error) {
	ticketsCount, _ := ticketResponse["totalInfracciones"].(float64)
	if ticketsCount == 0 {
		return nil, nil
	}
	rawTickets, _ := ticketResponse["infracciones"].([]interface{})

	var tickets []models.Ticket

	for _, rawTicket := range rawTickets {
		ticket, _ := rawTicket.(map[string]interface{})
		ticketNumber, _ := ticket["nroActa"].(string)
		amount, _ := ticket["importeTotal"].(float64)
		description := ""
		violations, _ := ticket["infracciones"].([]interface{})

		for _, rawViolation := range violations {
			if violation, ok := rawViolation.(map[string]interface{}); ok {
				violationDescription, _ := violation["descripcion"]
				if description == "" {
					description = fmt.Sprintf("%s", violationDescription)
				} else {
					description = fmt.Sprintf("%s / %s", description, violationDescription)
				}
			}
		}

		ticketDate, _ := ticket["fechaInfraccion"].(float64)
		ticketDueDate, _ := ticket["fechaVencimiento"].(float64)

		tickets = append(tickets, models.Ticket{
			Description:   description,
			AmountInCents: int64(amount * 100),
			TicketNumber:  ticketNumber,
			Date:          time.Unix(0, int64(time.Duration(ticketDate)*time.Millisecond)),
			DueDate:       time.Unix(0, int64(time.Duration(ticketDueDate)*time.Millisecond)),
		})
	}

	return tickets, nil
}

func (d *BA) getBARawTickets(plateNumber, captchaCode string) (map[string]interface{}, error) {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	req, err := http.NewRequest("GET", fmt.Sprintf(`%s?dominio=%s&reCaptcha=%s&cantPorPagina=10&paginaActual=1`, d.baseURL, plateNumber, captchaCode), nil)

	if err != nil {
		return nil, err
	}

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	rawTickets := make(map[string]interface{})
	err = json.NewDecoder(res.Body).Decode(&rawTickets)
	if err != nil {
		return nil, err
	}

	return rawTickets, nil
}

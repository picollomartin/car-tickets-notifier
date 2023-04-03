package districts

import (
	"net/http"
	"net/http/httputil"
	"strconv"

	"github.com/PuerkitoBio/goquery"

	"car-tickets-notifier/internal/models"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CaptchaSolver ...
type CaptchaSolver interface {
	GetCaptchaCode() (string, error)
}

// CABA ...
type CABA struct {
	captchaSolver CaptchaSolver
	baseURL       string
	apiURL        string
}

// Name ...
func (d *CABA) Name() string {
	return "Capital Federal"
}

type requestMetadata struct {
	captchaCode string
}

// GetTickets ...
func (d *CABA) GetTickets(plateNumber string) ([]models.Ticket, error) {
	metadata, err := d.getCABARequestMetadata()
	if err != nil {
		return nil, fmt.Errorf("fail getting request metadata: %w", err)
	}

	rawTickets, err := d.getCABARawTickets(plateNumber, metadata)
	if err != nil {
		return nil, fmt.Errorf("fail getting tickets from api: %w", err)
	}
	return parseCABARawTickets(rawTickets)
}

// NewCABA ...
func NewCABA(baseURL, apiURL string, captchaSolver CaptchaSolver) *CABA {
	return &CABA{
		captchaSolver: captchaSolver,
		baseURL:       baseURL,
		apiURL:        apiURL,
	}
}

func (d *CABA) getCABARequestMetadata() (*requestMetadata, error) {
	captchaCode, err := d.captchaSolver.GetCaptchaCode()
	if err != nil {
		return nil, fmt.Errorf("fail getting captcha code: %w", err)
	}
	return &requestMetadata{
		captchaCode: captchaCode,
	}, nil
}

func (d *CABA) getCABARawTickets(plateNumber string, metadata *requestMetadata) (string, error) {
	rawPayload := fmt.Sprintf(
		`tipo_consulta=Dominio&dominio=%s&tipo_doc=DNI&doc=&g-recaptcha-response=%s`,
		plateNumber, metadata.captchaCode)
	payload := strings.NewReader(rawPayload)

	client := &http.Client{}
	req, err := http.NewRequest("POST", d.apiURL, payload)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err := httputil.DumpResponse(res, true)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func parseCABARawTickets(rawTicketResponse string) ([]models.Ticket, error) {
	formattedHtml := strings.NewReader(rawTicketResponse)
	doc, err := goquery.NewDocumentFromReader(formattedHtml)
	if err != nil {
		return nil, fmt.Errorf("fail parsing ticket response into html: %w", err)
	}
	_, ok := doc.Find("[id=descarga_libre_deuda]").Attr("class")
	if ok { // No registered debt
		return nil, nil
	}
	var ticketsJSON []string

	doc.Find("input[type='checkbox'][name='actas[]']").Each(func(i int, selection *goquery.Selection) {
		jsonTicketData, ok := selection.Attr("data-json")
		if ok {
			ticketsJSON = append(ticketsJSON, jsonTicketData)
		} else { // We inject one manual check if json data is not present, happen when a ticket should be checked before payment
			ticketsJSON = append(ticketsJSON, fmt.Sprintf(`{
                "numeroActa":"manual-check-%d","fechaActa":"9999-12-31 23:59","montoActa":1,"fechaVencimiento":"31-12-9999","infracciones":[{"descripcion":"Verificación manual","lugar":"Desconocido"}]
			}`, i))
		}
	})

	var tickets []models.Ticket
	for _, ticketJSON := range ticketsJSON {
		rawTicket := make(map[string]interface{})
		err := json.Unmarshal([]byte(ticketJSON), &rawTicket)
		if err != nil {
			return nil, fmt.Errorf("fail unmarshalling ticket: %w", err)
		}
		ticket, err := mapRawTicket(rawTicket)
		if err != nil {
			return nil, fmt.Errorf("fail parsing raw ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}
	return tickets, nil
}

func mapRawTicket(rawTicket map[string]interface{}) (models.Ticket, error) {
	ticketNumber, _ := rawTicket["numeroActa"].(string)
	rawAmount, _ := rawTicket["montoActa"].(string)
	amount, err := strconv.Atoi(rawAmount)
	if err != nil {
		return models.Ticket{}, fmt.Errorf("parsing amount: %w of ticket: %s", err, ticketNumber)
	}
	ticketDate, _ := time.Parse("2006-01-02 15:04", rawTicket["fechaActa"].(string))
	description := ""
	violations, _ := rawTicket["infracciones"].([]interface{})

	for _, rawViolation := range violations {
		if violation, ok := rawViolation.(map[string]interface{}); ok {
			violationDescription, _ := violation["desc"]
			violationPlace, _ := violation["lugar"]
			if description == "" {
				description = fmt.Sprintf("%s - %s", violationPlace, violationDescription)
			} else {
				description = fmt.Sprintf("%s / %s - %s", description, violationPlace, violationDescription)
			}
		}
	}

	return models.Ticket{
		Description:   description,
		AmountInCents: int64(amount * 100), // to cents
		TicketNumber:  ticketNumber,
		Date:          ticketDate,
	}, nil
}

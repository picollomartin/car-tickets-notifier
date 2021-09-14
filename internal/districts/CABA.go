package districts

import (
	"net/http"

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
	sid         string
	token       string
	formBuildID string
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
	return d.lookupCABARequestMetadata(captchaCode)
}

func (d *CABA) lookupCABARequestMetadata(captchaCode string) (*requestMetadata, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", d.baseURL, nil)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("fail parsing body of CABA url for lookup metadata info: %w", err)
	}

	captchaSid, ok := doc.Find("[name=captcha_sid]").Attr("value")
	if !ok {
		return nil, fmt.Errorf("captcha_sid metadata not found")
	}

	captchaToken, ok := doc.Find("[name=captcha_token]").Attr("value")
	if !ok {
		return nil, fmt.Errorf("captcha_token metadata not found")
	}

	var formBuildID string
	doc.Find("[id=gcaba-infracciones-form]").Children().Each(func(i int, selection *goquery.Selection) {
		if attr, ok := selection.Attr("name"); ok && attr == "form_build_id" {
			formBuildID, _ = selection.Attr("value")
		}
	})
	if formBuildID == "" {
		return nil, fmt.Errorf("form_build_id metadata not found")
	}

	return &requestMetadata{
		captchaCode: captchaCode,
		sid:         captchaSid,
		token:       captchaToken,
		formBuildID: formBuildID,
	}, nil
}

func (d *CABA) getCABARawTickets(plateNumber string, metadata *requestMetadata) ([]map[string]interface{}, error) {
	rawPayload := fmt.Sprintf(
		`tipo_consulta=Dominio&dominio=%s&tipo_doc=DNI&doc=&form_build_id=%s&captcha_sid=%s&captcha_token=%s&captcha_response=Google+no+captcha&g-recaptcha-response=%s&form_id=gcaba_infracciones_form`,
		plateNumber, metadata.formBuildID, metadata.sid, metadata.token, metadata.captchaCode)
	payload := strings.NewReader(rawPayload)

	client := &http.Client{}
	req, err := http.NewRequest("POST", d.apiURL, payload)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var data []map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func parseCABARawTickets(rawTicketResponse []map[string]interface{}) ([]models.Ticket, error) {
	if len(rawTicketResponse) < 2 {
		return nil, fmt.Errorf("invalid ticket response: %v", rawTicketResponse)
	}
	formattedHtml := strings.NewReader(rawTicketResponse[1]["data"].(string))
	doc, err := goquery.NewDocumentFromReader(formattedHtml)
	if err != nil {
		return nil, fmt.Errorf("fail parsing ticket response into html: %w", err)
	}
	_, ok := doc.Find("[id=form-sin-deuda-submit]").Attr("value")
	if ok { // No registered debt
		return nil, nil
	}
	var ticketsJSON []string

	doc.Find(".accordionActasComprobantes").ChildrenFiltered(".panel-default").Each(func(i int, selection *goquery.Selection) {
		ticketSelector := selection.Find(".panel-title").ChildrenFiltered(".text-center").Children()
		jsonTicketData, ok := ticketSelector.Attr("data-json")
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
	amount, _ := rawTicket["montoActa"].(float64)
	ticketDate, _ := time.Parse("2006-01-02 15:04", rawTicket["fechaActa"].(string))
	dueDate, _ := time.Parse("02-01-2006", rawTicket["fechaVencimiento"].(string))
	description := ""
	violations, _ := rawTicket["infracciones"].([]interface{})

	for _, rawViolation := range violations {
		if violation, ok := rawViolation.(map[string]interface{}); ok {
			violationDescription, _ := violation["descripcion"]
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
		DueDate:       dueDate,
	}, nil
}

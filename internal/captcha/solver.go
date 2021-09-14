package captcha

import "github.com/2captcha/2captcha-go"

// Solver ...
type Solver struct {
	clientData api2captcha.ReCaptcha
	client     *api2captcha.Client
}

// NewSolver ...
func NewSolver(apiKey, siteKey, url string) *Solver {
	client := api2captcha.NewClient(apiKey)
	return &Solver{
		clientData: api2captcha.ReCaptcha{
			SiteKey: siteKey,
			Url:     url,
		},
		client: client,
	}
}

// GetCaptchaCode ...
func (s *Solver) GetCaptchaCode() (string, error) {
	return s.client.Solve(s.clientData.ToRequest())
}

package main

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"car-tickets-notifier/internal/captcha"
	"car-tickets-notifier/internal/districts"
	"car-tickets-notifier/internal/models"
	"car-tickets-notifier/internal/notifier"
	"car-tickets-notifier/internal/telegram"
	"strconv"
)

func main() {
	var cabaDistrictFlag, baDistrictFlag bool
	var plateNumber string
	rootCmd := &cobra.Command{
		Use: "car-tickets-notifier",
		Run: func(cmd *cobra.Command, args []string) {
			var districtList []models.District
			if cabaDistrictFlag {
				cabaDistrict, err := loadCABADistrict()
				if err != nil {
					cmd.PrintErrf("fail loading caba district: %s", err.Error())
					return
				}
				districtList = append(districtList, cabaDistrict)
			}
			if baDistrictFlag {
				baDistrict, err := loadBADistrict()
				if err != nil {
					cmd.PrintErrf("fail loading ba district: %s", err.Error())
					return
				}
				districtList = append(districtList, baDistrict)
			}

			chatID, _ := strconv.ParseInt(os.Getenv("TELEGRAM_CHAT_ID"), 10, 64)

			telegramBot, err := telegram.New(os.Getenv("TELEGRAM_BOT_TOKEN"), chatID)
			if err != nil {
				cmd.PrintErrf("fail loading telegram bot: %s", err.Error())
				return
			}

			notifier.Notify(telegramBot, plateNumber, districtList)

		},
	}
	rootCmd.Flags().BoolVar(&cabaDistrictFlag, "caba", true, "search tickets in CABA")
	rootCmd.Flags().BoolVar(&baDistrictFlag, "ba", true, "search tickets in BA")
	rootCmd.Flags().StringVarP(&plateNumber, "plateNumber", "p", "", "plate number related to tickets")
	rootCmd.MarkFlagRequired("plateNumber")

	rootCmd.Execute()
}

func loadBADistrict() (*districts.BA, error) {
	captchaSolverAPIKey := os.Getenv("CAPTCHA_SOLVER_API_KEY")
	captchaSolverSiteKey := os.Getenv("CAPTCHA_SOLVER_BA_SITE_KEY")
	baBaseURL := os.Getenv("BA_BASE_URL")
	baAPIURL := os.Getenv("BA_API_URL")
	if baAPIURL == "" || baBaseURL == "" || captchaSolverSiteKey == "" || captchaSolverAPIKey == "" {
		return nil, errors.New("missing ba env keys")
	}
	solver := captcha.NewSolver(captchaSolverAPIKey, captchaSolverSiteKey, baBaseURL)
	return districts.NewBA(baAPIURL, solver), nil
}

func loadCABADistrict() (*districts.CABA, error) {
	captchaSolverAPIKey := os.Getenv("CAPTCHA_SOLVER_API_KEY")
	captchaSolverSiteKey := os.Getenv("CAPTCHA_SOLVER_CABA_SITE_KEY")
	cabaBaseURL := os.Getenv("CABA_BASE_URL")
	cabaAPIURL := os.Getenv("CABA_API_URL")
	if captchaSolverAPIKey == "" || captchaSolverSiteKey == "" || cabaBaseURL == "" || cabaAPIURL == "" {
		return nil, errors.New("missing caba env keys")
	}
	solver := captcha.NewSolver(captchaSolverAPIKey, captchaSolverSiteKey, cabaBaseURL)

	return districts.NewCABA(cabaBaseURL, cabaAPIURL, solver), nil
}

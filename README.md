# 🔔 car-tickets-notifier

Personal script for get notifications (through telegram messages) about car tickets in some argentinian districts.
This uses [2captcha](https://2captcha.com/) because API information requires recaptcha verification in each district.

## 🔨 Build script

```
go build -o car-tickets-notifier cmd/notifier/main.go
```

## ⚙️ Environment

You will need some environment keys in order to run this script

```
CABA_BASE_URL= # Webpage of Capital Federal that reports car tickets (used for get captcha response)
CABA_API_URL= # API used by the CABA_BASE_URL (used for get tickets information)
BA_BASE_URL= # Webpage of Buenos Aires that reports car tickets (used for get captcha response)
BA_API_URL= # API used by the BA_API_URL (used for get tickets information)
CAPTCHA_SOLVER_BA_SITE_KEY= # Captcha site key of BA_BASE_URL
CAPTCHA_SOLVER_CABA_SITE_KEY= # Captcha site key of CABA_BASE_URL
CAPTCHA_SOLVER_API_KEY= # API key of 2captcha
TELEGRAM_BOT_TOKEN= # Telegram bot token
TELEGRAM_CHAT_ID= # User that would receive the car tickets report
```

## 🏃 Running

```
car-tickets-notifier --plateNumber ABC123
```

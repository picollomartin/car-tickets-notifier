package telegram

import (
	"math"
	"strings"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

const MaxMsgLength = 4096

// Telegram ...
type Telegram struct {
	client *tgbotapi.BotAPI
	chatID int64
}

// New ...
func New(token string, chatID int64) (*Telegram, error) {
	client, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Telegram{client: client, chatID: chatID}, nil
}

// Send ...
func (t *Telegram) Send(msg string) error {
	escapedMsg := escapeMessage(msg)
	batchs := int(math.Ceil(float64(len(escapedMsg)) / float64(MaxMsgLength)))

	for i := 0; i < batchs; i++ {
		batchStart := MaxMsgLength * i
		batchFinish := min(batchStart+MaxMsgLength, len(msg)-1)
		_, err := t.client.Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:           t.chatID,
				ReplyToMessageID: 0,
			},
			Text:                  escapeMessage(msg[batchStart:batchFinish]),
			DisableWebPagePreview: false,
			ParseMode:             "MarkdownV2",
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func min(i int, i2 int) int {
	return int(math.Min(float64(i), float64(i2)))
}

func escapeMessage(msg string) string {
	replacer := strings.NewReplacer(
		".", "\\.",
		")", "\\)",
		"(", "\\(",
		"-", "\\-",
		"Ñ", "n",
		"ñ", "n")

	return strings.ToValidUTF8(replacer.Replace(msg), "")
}

package dispatcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/amir20/dozzle/types"
)

type TelegramDispatcher struct {
	TelegramMessage  string
	TelegramBotToken string
	TelegramChatId   int
	template         *template.Template
}

func NewTelegramDispatcher(message string, botToken string, chatId int) (*TelegramDispatcher, error) {
	// Template par défaut si aucun message fourni
	if message == "" {
		message = "🚨 <b>{{.Container.Name}}</b>\n" +
			"Type: {{.Type}}\n" +
			"Detail: {{.Detail}}\n" +
			"Host: {{.Container.HostName}}\n" +
			"Time: {{.Timestamp.Format \"2006-01-02 15:04:05\"}}"
	}

	tmpl, err := template.New("telegram").Parse(message)
	if err != nil {
		return nil, fmt.Errorf("failed to parse telegram template: %w", err)
	}

	return &TelegramDispatcher{
		TelegramMessage:  message,
		TelegramBotToken: botToken,
		TelegramChatId:   chatId,
		template:         tmpl,
	}, nil
}

func (t *TelegramDispatcher) Send(ctx context.Context, notification types.Notification) error {
	result := t.SendTest(ctx, notification)
	if !result.Success {
		return fmt.Errorf("webhook notification failed: %s", result.Error)
	}
	return nil
}

func (t *TelegramDispatcher) SendTest(ctx context.Context, notification types.Notification) TestResult {
	// Rendre le template avec les données de la notification
	var rendered bytes.Buffer
	if err := t.template.Execute(&rendered, notification); err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to render template: %v", err)}
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.TelegramBotToken)
	payload := map[string]interface{}{
		"chat_id":    t.TelegramChatId,
		"text":       rendered.String(),
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to marshal telegram payload: %v", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to create request: %v", err)}
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return TestResult{Success: false, Error: fmt.Sprintf("failed to send telegram message: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return TestResult{Success: false, Error: fmt.Sprintf("telegram API returned status %d", resp.StatusCode)}
	}

	return TestResult{Success: true}
}

package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kjannette/trahn-backend/internal/httputil"
)

type Sender struct {
	webhookURL string
	botName    string
	httpClient *http.Client
	retry      httputil.RetryConfig
}

func NewSender(webhookURL, botName string) *Sender {
	if botName == "" {
		botName = "TrahnGridTrader"
	}
	return &Sender{
		webhookURL: webhookURL,
		botName:    botName,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		retry: httputil.RetryConfig{
			MaxAttempts: 3,
			BaseDelay:   1 * time.Second,
			MaxDelay:    5 * time.Second,
		},
	}
}

func (s *Sender) Send(msg string) {
	formatted := fmt.Sprintf("[%s] %s", s.botName, msg)
	fmt.Printf("[%s] %s\n", time.Now().UTC().Format(time.RFC3339), formatted)

	if s.webhookURL == "" {
		return
	}

	payload := s.formatPayload(formatted)
	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[CHAT ERROR] marshal: %v\n", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := httputil.Do(ctx, s.httpClient, s.retry, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		fmt.Printf("[CHAT ERROR] Failed to send notification after retries: %v\n", err)
		return
	}
	resp.Body.Close()
}

func (s *Sender) formatPayload(msg string) map[string]string {
	if strings.Contains(s.webhookURL, "discord") {
		return map[string]string{
			"content":  msg,
			"username": s.botName,
		}
	}
	return map[string]string{
		"text":     fmt.Sprintf("`%s`", msg),
		"username": s.botName,
	}
}

func (s *Sender) Enabled() bool {
	return s.webhookURL != ""
}

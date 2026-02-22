package notifications

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend_NoWebhook(t *testing.T) {
	s := NewSender("", "TestBot")
	if s.Enabled() {
		t.Fatal("should not be enabled with empty URL")
	}
	// Should log to console without error
	s.Send("hello from test")
	t.Log("Send with no webhook: OK (console only)")
}

func TestSend_SlackFormat(t *testing.T) {
	var received map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := NewSender(srv.URL, "TestBot")
	if !s.Enabled() {
		t.Fatal("should be enabled")
	}

	s.Send("grid recalculated")

	if received["username"] != "TestBot" {
		t.Fatalf("username: got %s", received["username"])
	}
	if received["text"] == "" {
		t.Fatal("text should not be empty")
	}
	t.Logf("Slack payload: %+v", received)
}

func TestSend_DiscordFormat(t *testing.T) {
	var received map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// URL containing "discord" triggers Discord format
	s := NewSender(srv.URL+"/discord/webhook", "TrahnBot")
	s.Send("trade executed: buy 0.04 ETH @ $2600")

	if received["content"] == "" {
		t.Fatal("content should not be empty for Discord")
	}
	if received["username"] != "TrahnBot" {
		t.Fatalf("username: got %s", received["username"])
	}
	if _, hasText := received["text"]; hasText {
		t.Fatal("Discord payload should not have 'text' field")
	}
	t.Logf("Discord payload: %+v", received)
}

func TestSend_WebhookError(t *testing.T) {
	s := NewSender("http://localhost:1/bogus", "TestBot")
	// Should not panic, just log the error
	s.Send("this will fail gracefully")
	t.Log("Webhook error handled gracefully")
}

func TestDefaultBotName(t *testing.T) {
	s := NewSender("", "")
	if s.botName != "TrahnGridTrader" {
		t.Fatalf("expected default bot name, got %s", s.botName)
	}
}

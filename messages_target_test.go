package kappelas

import (
	"encoding/json"
	"strings"
	"testing"
)

// Verifies the chat_id | user_id recipient rule and action_button serialise correctly.

func TestSendMessageParams_UserID(t *testing.T) {
	b, _ := json.Marshal(SendMessageParams{UserID: "uuid-1", Text: "hi"})
	s := string(b)
	if !strings.Contains(s, `"user_id":"uuid-1"`) {
		t.Fatalf("user_id missing: %s", s)
	}
	if strings.Contains(s, "chat_id") {
		t.Fatalf("chat_id should be omitted when UserID set: %s", s)
	}
}

func TestSendMessageParams_ChatID(t *testing.T) {
	b, _ := json.Marshal(SendMessageParams{ChatID: 42, Text: "hi"})
	s := string(b)
	if !strings.Contains(s, `"chat_id":42`) {
		t.Fatalf("chat_id missing: %s", s)
	}
	if strings.Contains(s, "user_id") {
		t.Fatalf("user_id should be omitted when ChatID set: %s", s)
	}
}

func TestSendMessageParams_ActionButton(t *testing.T) {
	b, _ := json.Marshal(SendMessageParams{
		ChatID: 42, Text: "hi",
		ActionButton: &ActionButton{Label: "Copy", Type: "copy_text", Value: "123"},
	})
	if !strings.Contains(string(b), `"action_button":{"label":"Copy","type":"copy_text","value":"123"}`) {
		t.Fatalf("action_button wrong: %s", b)
	}
}

func TestRecipientParams_UserID(t *testing.T) {
	cases := []struct {
		name string
		v    any
	}{
		{"edit", EditMessageParams{UserID: "u", MessageID: 1, NewText: "x"}},
		{"delete", DeleteMessageParams{UserID: "u", MessageID: 1}},
		{"typing", SendTypingParams{UserID: "u"}},
		{"carousel", SendCarouselParams{UserID: "u", Carousel: []CarouselCard{{ID: "p1", Title: "A"}}}},
	}
	for _, tc := range cases {
		b, _ := json.Marshal(tc.v)
		if !strings.Contains(string(b), `"user_id":"u"`) {
			t.Fatalf("%s: user_id missing: %s", tc.name, b)
		}
		if strings.Contains(string(b), `"chat_id"`) {
			t.Fatalf("%s: chat_id should be omitted: %s", tc.name, b)
		}
	}
}

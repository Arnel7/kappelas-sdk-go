package kappelas

import (
	"encoding/json"
	"sync"
	"time"
)

// ─── Options ─────────────────────────────────────────────────────────────────

// BotOption configures a Bot.
type BotOption func(*botConfig)

type botConfig struct {
	baseURL      string
	maxRetries   int
	timeout      time.Duration
	wsMaxRetries int
}

// WithBaseURL overrides the API base URL (default: https://api.kappelas.com).
func WithBaseURL(u string) BotOption {
	return func(c *botConfig) { c.baseURL = u }
}

// WithMaxRetries sets the maximum number of HTTP retry attempts (default: 2).
func WithMaxRetries(n int) BotOption {
	return func(c *botConfig) { c.maxRetries = n }
}

// WithTimeout sets the HTTP request timeout (default: 30s).
func WithTimeout(d time.Duration) BotOption {
	return func(c *botConfig) { c.timeout = d }
}

// WithWSMaxRetries sets the maximum WebSocket reconnect attempts (default: 12).
func WithWSMaxRetries(n int) BotOption {
	return func(c *botConfig) { c.wsMaxRetries = n }
}

// ─── Bot ─────────────────────────────────────────────────────────────────────

// Bot is the Kappela bot client. Authenticate with a token from BotMother.
//
// Example:
//
//	bot := kappelas.NewBot("YOUR_BOT_TOKEN")
//
//	bot.OnMessage(func(msg *kappelas.Message) {
//	    bot.Messages.Send(ctx, kappelas.SendMessageParams{
//	        ChatID: msg.ChatID,
//	        Text:   "Echo: " + *msg.Text,
//	    })
//	})
//
//	bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
//	    bot.Messages.Send(ctx, kappelas.SendMessageParams{
//	        ChatID: cb.ChatID,
//	        Text:   "Button clicked: " + cb.CallbackData,
//	    })
//	})
//
//	bot.Start()
//	select {} // keep alive
type Bot struct {
	// Messages provides methods to send and manage messages.
	Messages *MessagesResource
	// Chats provides methods to list and iterate over chats.
	Chats *ChatsResource
	// Webhooks provides methods to manage webhooks.
	Webhooks *WebhooksResource
	// Profile provides access to the bot's own profile.
	Profile *BotProfileResource

	http *httpClient
	ws   *wsClient

	mu          sync.RWMutex
	msgHandlers []func(*Message)
	cbHandlers  []func(*CallbackQuery)
	connHs      []func()
	disconnHs   []func(int, string)
	errHs       []func(error)
}

// NewBot creates a Bot authenticated with the given token.
func NewBot(token string, opts ...BotOption) *Bot {
	cfg := &botConfig{
		baseURL:      defaultBase,
		maxRetries:   2,
		timeout:      30 * time.Second,
		wsMaxRetries: 12,
	}
	for _, o := range opts {
		o(cfg)
	}

	base := "/v1/" + token
	h := newHTTPClient(cfg.baseURL, cfg.maxRetries, cfg.timeout)
	w := newWSClient(toWSURL(cfg.baseURL, base+"/ws"), cfg.wsMaxRetries)

	b := &Bot{
		http:     h,
		ws:       w,
		Messages: &MessagesResource{http: h, base: base},
		Chats:    &ChatsResource{http: h, base: base},
		Webhooks: &WebhooksResource{http: h, base: base},
		Profile:  &BotProfileResource{http: h, base: base},
	}

	w.onRaw = func(data []byte) { b.dispatchWire(data) }
	w.onConnected = func() {
		b.mu.RLock()
		hs := append([]func(){}, b.connHs...)
		b.mu.RUnlock()
		for _, h := range hs {
			h()
		}
	}
	w.onDisconnected = func(code int, reason string) {
		b.mu.RLock()
		hs := append([]func(int, string){}, b.disconnHs...)
		b.mu.RUnlock()
		for _, h := range hs {
			h(code, reason)
		}
	}
	w.onError = func(err error) {
		b.mu.RLock()
		hs := append([]func(error){}, b.errHs...)
		b.mu.RUnlock()
		for _, h := range hs {
			h(err)
		}
	}

	return b
}

// ─── Handler registration ────────────────────────────────────────────────────

// OnMessage registers a handler called for every incoming message.
func (b *Bot) OnMessage(h func(*Message)) {
	b.mu.Lock()
	b.msgHandlers = append(b.msgHandlers, h)
	b.mu.Unlock()
}

// OnCallbackQuery registers a handler called when a user clicks an inline button.
func (b *Bot) OnCallbackQuery(h func(*CallbackQuery)) {
	b.mu.Lock()
	b.cbHandlers = append(b.cbHandlers, h)
	b.mu.Unlock()
}

// OnConnected registers a handler called when the WebSocket connects (or reconnects).
func (b *Bot) OnConnected(h func()) {
	b.mu.Lock()
	b.connHs = append(b.connHs, h)
	b.mu.Unlock()
}

// OnDisconnected registers a handler called when the WebSocket disconnects.
func (b *Bot) OnDisconnected(h func(code int, reason string)) {
	b.mu.Lock()
	b.disconnHs = append(b.disconnHs, h)
	b.mu.Unlock()
}

// OnError registers a handler called on WebSocket or connection errors.
func (b *Bot) OnError(h func(error)) {
	b.mu.Lock()
	b.errHs = append(b.errHs, h)
	b.mu.Unlock()
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────

// Start connects via WebSocket and begins receiving events in the background.
// Your OnMessage and OnCallbackQuery handlers will be called for each event.
func (b *Bot) Start() {
	b.ws.start()
}

// Stop closes the WebSocket connection.
func (b *Bot) Stop() {
	b.ws.stop()
}

// Connected reports whether the WebSocket is currently open.
func (b *Bot) Connected() bool {
	return b.ws.connected()
}

// HandleWebhook processes a webhook payload sent by Kappela to your server.
// Call this inside your HTTP handler and respond 200 immediately.
// The same OnMessage and OnCallbackQuery handlers fire for both WS and webhook events.
//
// Example (net/http):
//
//	http.HandleFunc("/kappela-webhook", func(w http.ResponseWriter, r *http.Request) {
//	    body, _ := io.ReadAll(r.Body)
//	    bot.HandleWebhook(body)
//	    w.WriteHeader(http.StatusOK)
//	})
func (b *Bot) HandleWebhook(body []byte) {
	dispatchWebhook(body, b.dispatchMessage, b.dispatchCallback)
}

// ─── Internal dispatch ───────────────────────────────────────────────────────

func (b *Bot) dispatchWire(data []byte) {
	var ev struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return
	}
	switch ev.Type {
	case "message":
		var msg Message
		if json.Unmarshal(ev.Data, &msg) == nil {
			b.dispatchMessage(&msg)
		}
	case "callback_query", "callback":
		var cb CallbackQuery
		if json.Unmarshal(ev.Data, &cb) == nil {
			b.dispatchCallback(&cb)
		}
	}
}

func (b *Bot) dispatchMessage(msg *Message) {
	b.mu.RLock()
	hs := append([]func(*Message){}, b.msgHandlers...)
	b.mu.RUnlock()
	for _, h := range hs {
		h(msg)
	}
}

func (b *Bot) dispatchCallback(cb *CallbackQuery) {
	b.mu.RLock()
	hs := append([]func(*CallbackQuery){}, b.cbHandlers...)
	b.mu.RUnlock()
	for _, h := range hs {
		h(cb)
	}
}

// ─── Shared webhook parser ───────────────────────────────────────────────────

var webhookMessageTypes = map[string]bool{
	"text": true, "image": true, "video": true, "audio": true, "document": true,
	"system": true, "poll": true, "sticker": true, "location": true, "contact": true,
}

// dispatchWebhook parses a flat webhook payload and calls the appropriate dispatcher.
func dispatchWebhook(body []byte, onMsg func(*Message), onCB func(*CallbackQuery)) {
	var p struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return
	}

	switch p.Type {
	case "callback":
		var cb CallbackQuery
		if json.Unmarshal(body, &cb) == nil {
			onCB(&cb)
		}
	default:
		if !webhookMessageTypes[p.Type] {
			return
		}
		var raw struct {
			MessageID int64           `json:"message_id"`
			ChatID    int64           `json:"chat_id"`
			SenderID  *string         `json:"sender_id"`
			Type      MessageType     `json:"type"`
			Text      *string         `json:"text"`
			ExtraData json.RawMessage `json:"extra_data"`
			SentAt    int64           `json:"sent_at"`
		}
		if json.Unmarshal(body, &raw) != nil {
			return
		}
		onMsg(&Message{
			ID:        raw.MessageID,
			ChatID:    raw.ChatID,
			SenderID:  raw.SenderID,
			Type:      raw.Type,
			Text:      raw.Text,
			ExtraData: raw.ExtraData,
			Status:    MessageStatusSent,
			CreatedAt: raw.SentAt,
			Mentions:  []string{},
		})
	}
}

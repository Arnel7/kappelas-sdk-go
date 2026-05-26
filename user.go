package kappelas

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// UserOption configures a User.
type UserOption func(*userConfig)

type userConfig struct {
	baseURL      string
	maxRetries   int
	timeout      time.Duration
	wsMaxRetries int
}

// WithUserBaseURL overrides the API base URL for a User client.
func WithUserBaseURL(u string) UserOption {
	return func(c *userConfig) { c.baseURL = u }
}

// WithUserMaxRetries sets the maximum number of HTTP retry attempts for a User client.
func WithUserMaxRetries(n int) UserOption {
	return func(c *userConfig) { c.maxRetries = n }
}

// WithUserTimeout sets the HTTP request timeout for a User client.
func WithUserTimeout(d time.Duration) UserOption {
	return func(c *userConfig) { c.timeout = d }
}

// WithUserWSMaxRetries sets the maximum WebSocket reconnect attempts for a User client.
func WithUserWSMaxRetries(n int) UserOption {
	return func(c *userConfig) { c.wsMaxRetries = n }
}

// User is the Kappela personal automation client. Authenticate with a personal
// API key (sk_...) to send messages and receive events as yourself.
//
// Example:
//
//	me := kappelas.NewUser("sk_...")
//
//	me.OnMessage(func(msg *kappelas.Message) {
//	    fmt.Println("New message:", msg.Text)
//	})
//
//	me.Start()
//	select {}
type User struct {
	// Messages provides methods to send and manage messages.
	Messages *MessagesResource
	// Chats provides methods to list and iterate over chats.
	Chats *ChatsResource
	// Webhooks provides methods to manage webhooks.
	Webhooks *WebhooksResource
	// Profile provides access to your own profile.
	Profile *UserProfileResource

	http *httpClient
	ws   *wsClient

	mu          sync.RWMutex
	msgHandlers []func(*Message)
	cbHandlers  []func(*CallbackQuery)
	connHs      []func()
	disconnHs   []func(int, string)
	errHs       []func(error)
}

// NewUser creates a User authenticated with the given personal API key.
func NewUser(apiKey string, opts ...UserOption) *User {
	cfg := &userConfig{
		baseURL:      defaultBase,
		maxRetries:   2,
		timeout:      30 * time.Second,
		wsMaxRetries: 12,
	}
	for _, o := range opts {
		o(cfg)
	}

	const base = "/v1/me"
	h := newHTTPClient(cfg.baseURL, cfg.maxRetries, cfg.timeout)
	h.setAuth(map[string]string{"X-Api-Key": apiKey})

	wsPath := fmt.Sprintf("%s/ws?api_key=%s", base, apiKey)
	w := newWSClient(toWSURL(cfg.baseURL, wsPath), cfg.wsMaxRetries)

	u := &User{
		http:     h,
		ws:       w,
		Messages: &MessagesResource{http: h, base: base},
		Chats:    &ChatsResource{http: h, base: base},
		Webhooks: &WebhooksResource{http: h, base: base},
		Profile:  &UserProfileResource{http: h, base: base},
	}

	w.onRaw = func(data []byte) { u.dispatchWire(data) }
	w.onConnected = func() {
		u.mu.RLock()
		hs := append([]func(){}, u.connHs...)
		u.mu.RUnlock()
		for _, h := range hs {
			h()
		}
	}
	w.onDisconnected = func(code int, reason string) {
		u.mu.RLock()
		hs := append([]func(int, string){}, u.disconnHs...)
		u.mu.RUnlock()
		for _, h := range hs {
			h(code, reason)
		}
	}
	w.onError = func(err error) {
		u.mu.RLock()
		hs := append([]func(error){}, u.errHs...)
		u.mu.RUnlock()
		for _, h := range hs {
			h(err)
		}
	}

	return u
}

// ─── Handler registration ────────────────────────────────────────────────────

// OnMessage registers a handler called for every incoming message.
func (u *User) OnMessage(h func(*Message)) {
	u.mu.Lock()
	u.msgHandlers = append(u.msgHandlers, h)
	u.mu.Unlock()
}

// OnCallbackQuery registers a handler called when a user clicks an inline button.
func (u *User) OnCallbackQuery(h func(*CallbackQuery)) {
	u.mu.Lock()
	u.cbHandlers = append(u.cbHandlers, h)
	u.mu.Unlock()
}

// OnConnected registers a handler called when the WebSocket connects (or reconnects).
func (u *User) OnConnected(h func()) {
	u.mu.Lock()
	u.connHs = append(u.connHs, h)
	u.mu.Unlock()
}

// OnDisconnected registers a handler called when the WebSocket disconnects.
func (u *User) OnDisconnected(h func(code int, reason string)) {
	u.mu.Lock()
	u.disconnHs = append(u.disconnHs, h)
	u.mu.Unlock()
}

// OnError registers a handler called on WebSocket or connection errors.
func (u *User) OnError(h func(error)) {
	u.mu.Lock()
	u.errHs = append(u.errHs, h)
	u.mu.Unlock()
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────

// Start connects via WebSocket and begins receiving events in the background.
func (u *User) Start() {
	u.ws.start()
}

// Stop closes the WebSocket connection.
func (u *User) Stop() {
	u.ws.stop()
}

// Connected reports whether the WebSocket is currently open.
func (u *User) Connected() bool {
	return u.ws.connected()
}

// HandleWebhook processes a webhook payload sent by Kappela to your server.
//
// Example (net/http):
//
//	http.HandleFunc("/kappela-webhook", func(w http.ResponseWriter, r *http.Request) {
//	    body, _ := io.ReadAll(r.Body)
//	    me.HandleWebhook(body)
//	    w.WriteHeader(http.StatusOK)
//	})
func (u *User) HandleWebhook(body []byte) {
	dispatchWebhook(body, u.dispatchMessage, u.dispatchCallback)
}

// ─── Internal dispatch ───────────────────────────────────────────────────────

func (u *User) dispatchWire(data []byte) {
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
			u.dispatchMessage(&msg)
		}
	case "callback_query", "callback":
		var cb CallbackQuery
		if json.Unmarshal(ev.Data, &cb) == nil {
			u.dispatchCallback(&cb)
		}
	}
}

func (u *User) dispatchMessage(msg *Message) {
	u.mu.RLock()
	hs := append([]func(*Message){}, u.msgHandlers...)
	u.mu.RUnlock()
	for _, h := range hs {
		h(msg)
	}
}

func (u *User) dispatchCallback(cb *CallbackQuery) {
	u.mu.RLock()
	hs := append([]func(*CallbackQuery){}, u.cbHandlers...)
	u.mu.RUnlock()
	for _, h := range hs {
		h(cb)
	}
}

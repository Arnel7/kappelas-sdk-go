package kappelas

import (
	"fmt"
	"net/url"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var maskAPIKey = regexp.MustCompile(`([?&]api_key=)[^&]+`)

type wsClient struct {
	rawURL     string
	displayURL string
	maxRetries int

	mu   sync.Mutex
	conn *websocket.Conn
	quit chan struct{}

	onRaw          func([]byte)
	onConnected    func()
	onDisconnected func(code int, reason string)
	onError        func(error)
}

func newWSClient(rawURL string, maxRetries int) *wsClient {
	return &wsClient{
		rawURL:     rawURL,
		displayURL: maskAPIKey.ReplaceAllString(rawURL, "${1}***"),
		maxRetries: maxRetries,
	}
}

// start connects to the WebSocket and begins receiving messages in a goroutine.
// Calling start() on an already-running client is a no-op.
func (w *wsClient) start() {
	w.mu.Lock()
	if w.quit != nil {
		w.mu.Unlock()
		return
	}
	w.quit = make(chan struct{})
	w.mu.Unlock()
	go w.loop()
}

func (w *wsClient) loop() {
	attempts := 0
	for {
		select {
		case <-w.quit:
			return
		default:
		}

		if attempts > 0 {
			if attempts >= w.maxRetries {
				if w.onError != nil {
					w.onError(fmt.Errorf("websocket: max reconnect attempts (%d) reached", w.maxRetries))
				}
				return
			}
			delay := time.Duration(min(1<<uint(attempts-1), 30)) * time.Second
			select {
			case <-w.quit:
				return
			case <-time.After(delay):
			}
		}

		conn, _, err := websocket.DefaultDialer.Dial(w.rawURL, nil)
		if err != nil {
			if w.onError != nil {
				w.onError(fmt.Errorf("websocket (%s): %w", w.displayURL, err))
			}
			attempts++
			continue
		}

		w.mu.Lock()
		w.conn = conn
		w.mu.Unlock()
		attempts = 0

		if w.onConnected != nil {
			w.onConnected()
		}

		code, reason := w.readLoop(conn)

		w.mu.Lock()
		w.conn = nil
		w.mu.Unlock()

		if w.onDisconnected != nil {
			w.onDisconnected(code, reason)
		}

		select {
		case <-w.quit:
			return
		default:
			attempts++
		}
	}
}

func (w *wsClient) readLoop(conn *websocket.Conn) (code int, reason string) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if ce, ok := err.(*websocket.CloseError); ok {
				return ce.Code, ce.Text
			}
			return 0, err.Error()
		}
		if w.onRaw != nil {
			w.onRaw(msg)
		}
	}
}

// stop closes the WebSocket connection and stops the reconnect loop.
// Safe to call multiple times or before start().
func (w *wsClient) stop() {
	w.mu.Lock()
	q := w.quit
	w.quit = nil
	conn := w.conn
	w.mu.Unlock()

	if q != nil {
		close(q)
	}
	if conn != nil {
		conn.Close()
	}
}

// connected reports whether the WebSocket connection is currently open.
func (w *wsClient) connected() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn != nil
}

// ─── URL helper ──────────────────────────────────────────────────────────────

func toWSURL(httpURL, path string) string {
	base, err := url.Parse(httpURL)
	if err != nil {
		return path
	}
	ref, err := url.Parse(path)
	if err != nil {
		return path
	}
	u := base.ResolveReference(ref)
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	default:
		u.Scheme = "ws"
	}
	return u.String()
}

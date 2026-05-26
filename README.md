# kappelas-sdk-go

[![Go Reference](https://pkg.go.dev/badge/github.com/kappelas/kappelas-sdk-go.svg)](https://pkg.go.dev/github.com/kappelas/kappelas-sdk-go)
[![Go version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![GitHub](https://img.shields.io/badge/GitHub-source-181717?logo=github)](https://github.com/Arnel7/kappelas-sdk-go)

**Official Go SDK for the [Kappela](https://kappelas.com) messaging platform.**  
Build bots and personal automations — send messages, handle events, manage chats.

---

## Table of contents

- [Prerequisites](#prerequisites)
- [Install](#install)
- [Quick start](#quick-start)
- [Events — WebSocket vs Webhook](#events--websocket-vs-webhook)
- [API reference](#api-reference)
  - [messages](#messages)
  - [chats](#chats)
  - [webhooks](#webhooks)
  - [profile](#profile)
- [Keyboards](#keyboards)
- [Error handling](#error-handling)
- [File input](#file-input)

---

## Prerequisites

You need a bot token from **BotMother**, the official Kappela bot manager.

1. Open Kappela and start a conversation with [**BotMother**](https://kappelas.com/bot/botmother_bot)
2. Follow the instructions to create a bot
3. BotMother gives you a token — keep it secret, it gives full control over your bot

For personal automation (sending messages as yourself), generate an API key from your Kappela account settings (`sk_...`).

---

## Install

```bash
go get github.com/kappelas/kappelas-sdk-go
```

Requires **Go 1.21+**.

---

## Quick start

### Bot

```go
package main

import (
    "context"
    "fmt"

    "github.com/kappelas/kappelas-sdk-go"
)

func main() {
    bot := kappelas.NewBot("YOUR_BOT_TOKEN")

    bot.OnMessage(func(msg *kappelas.Message) {
        ctx := context.Background()
        bot.Messages.Send(ctx, kappelas.SendMessageParams{
            ChatID: msg.ChatID,
            Text:   "Echo: " + *msg.Text,
        })
    })

    bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
        ctx := context.Background()
        bot.Messages.Send(ctx, kappelas.SendMessageParams{
            ChatID: cb.ChatID,
            Text:   "You clicked: " + cb.CallbackData,
        })
    })

    bot.Start()
    select {} // keep alive
}
```

### Personal automation

```go
me := kappelas.NewUser("sk_...")

me.OnMessage(func(msg *kappelas.Message) {
    if msg.Text != nil {
        fmt.Printf("[%d] %s\n", msg.ChatID, *msg.Text)
    }
})

me.Start()
select {}
```

---

## Events — WebSocket vs Webhook

| Mode | Method | Best for |
|------|--------|----------|
| **WebSocket** | `bot.Start()` | Development, local scripts |
| **Webhook** | `bot.Webhooks.Set()` + `bot.HandleWebhook()` | Production servers |

The same `OnMessage` and `OnCallbackQuery` handlers work in both modes — no code change needed when switching.

### WebSocket (development)

```go
bot := kappelas.NewBot("YOUR_BOT_TOKEN")

bot.OnMessage(func(msg *kappelas.Message) { /* ... */ })
bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) { /* ... */ })

bot.Start()   // non-blocking, auto-reconnects on disconnect
select {}
```

### Webhook (production)

```go
import (
    "io"
    "net/http"

    "github.com/kappelas/kappelas-sdk-go"
)

bot := kappelas.NewBot("YOUR_BOT_TOKEN")

// register once
ctx := context.Background()
bot.Webhooks.Set(ctx, kappelas.SetWebhookParams{
    URL: "https://your-server.com/kappela-webhook",
})

bot.OnMessage(func(msg *kappelas.Message) { /* ... */ })
bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) { /* ... */ })

http.HandleFunc("/kappela-webhook", func(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    bot.HandleWebhook(body)
    w.WriteHeader(http.StatusOK)
})

http.ListenAndServe(":8080", nil)
```

> Do **not** call `bot.Start()` in webhook mode.

### Event reference

| Method | Signature | Description |
|--------|-----------|-------------|
| `OnMessage` | `func(*Message)` | Incoming message of any type |
| `OnCallbackQuery` | `func(*CallbackQuery)` | Inline button clicked by a user |
| `OnConnected` | `func()` | WebSocket connected or reconnected |
| `OnDisconnected` | `func(code int, reason string)` | WebSocket disconnected |
| `OnError` | `func(error)` | Connection or transport error |

### `CallbackQuery` fields

```go
bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
    cb.ChatID          // int64   — chat where the button was clicked
    cb.SenderID        // string  — UUID of the user who clicked
    cb.SenderNom       // *string — display name (e.g. "Arnel LAWSON")
    cb.SenderUsername  // *string — username (e.g. "arnell")
    cb.CallbackData    // string  — value set on the button
    cb.SentAt          // int64   — Unix timestamp (seconds)
})
```

> Clicks are deduplicated server-side — your handler fires exactly once per click.

---

## API reference

### Constructor options

#### `kappelas.NewBot(token string, opts ...BotOption) *Bot`

| Option | Default | Description |
|--------|---------|-------------|
| `WithBaseURL(url)` | `https://api.kappelas.com` | Override API base URL |
| `WithMaxRetries(n)` | `2` | HTTP retry count on 429 / 5xx |
| `WithTimeout(d)` | `30s` | Per-request timeout |
| `WithWSMaxRetries(n)` | `12` | Max WebSocket reconnect attempts |

#### `kappelas.NewUser(apiKey string, opts ...UserOption) *User`

| Option | Default | Description |
|--------|---------|-------------|
| `WithUserBaseURL(url)` | `https://api.kappelas.com` | Override API base URL |
| `WithUserMaxRetries(n)` | `2` | HTTP retry count on 429 / 5xx |
| `WithUserTimeout(d)` | `30s` | Per-request timeout |
| `WithUserWSMaxRetries(n)` | `12` | Max WebSocket reconnect attempts |

```go
bot := kappelas.NewBot("token",
    kappelas.WithTimeout(10 * time.Second),
    kappelas.WithMaxRetries(3),
)
```

---

### `messages`

#### `Messages.Send(ctx, params)` → `(*SendResult, error)`

```go
yes, no := "yes", "no"
replyTo := int64(123) // optional — ID of the message to reply to

result, err := bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID:    42,
    Text:      "Hello!",
    ReplyToID: &replyTo,
    ReplyMarkup: kappelas.InlineKeyboard{
        InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
            {Text: "Yes", CallbackData: &yes},
            {Text: "No",  CallbackData: &no},
        }},
    },
})
// → &SendResult{MessageID: ..., CreatedAt: ...}
```

#### `Messages.SendPhoto(ctx, params)` → `(*SendMediaResult, error)`

```go
data, _ := os.ReadFile("banner.png")
result, err := bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
    ChatID:  42,
    File:    kappelas.FileInput{Data: data, Filename: "banner.png", ContentType: "image/png"},
    Caption: "Check this out!",
})
// → SendMediaResult{MessageID, CreatedAt, MediaID}
```

#### `Messages.SendVideo` / `SendDocument` / `SendAudio` → `(*SendMediaResult, error)`

Same shape — pass the appropriate `FileInput`.

#### `Messages.SendCarousel(ctx, params)` → `(*SendCarouselResult, error)`

```go
btn := "Buy"
bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
    ChatID: 42,
    Text:   "Pick a product:",
    Carousel: []kappelas.CarouselCard{
        {ID: "p1", Title: "Widget A", ButtonText: &btn},
        {ID: "p2", Title: "Widget B", ButtonText: &btn},
    },
    QuickReplyButtons: []string{"See more", "Cancel"},
})
```

#### `Messages.Edit(ctx, params)` → `(*EditMessageResult, error)`

```go
// Edit text
bot.Messages.Edit(ctx, kappelas.EditMessageParams{
    ChatID: 42, MessageID: 123, NewText: "Updated!",
})

// Edit inline keyboard only
import "encoding/json"
done := "done"
kb, _ := json.Marshal(kappelas.InlineKeyboard{
    InlineKeyboard: [][]kappelas.InlineKeyboardButton{
        {{Text: "Done ✅", CallbackData: &done}},
    },
})
bot.Messages.Edit(ctx, kappelas.EditMessageParams{
    ChatID: 42, MessageID: 123, NewExtraData: kb,
})
// → EditMessageResult{Edited: true, MessageID: 123}
```

#### `Messages.SendTyping(ctx, params)` → `(*TypingResult, error)`

```go
bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: 42})                       // show (default)
f := false
bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: 42, IsTyping: &f})         // hide
```

#### `Messages.Delete(ctx, params)` → `(*DeleteResult, error)`

```go
bot.Messages.Delete(ctx, kappelas.DeleteMessageParams{ChatID: 42, MessageID: 123})
// → DeleteResult{Deleted: true}
```

---

### `chats`

#### `Chats.List(ctx, params)` → `(ChatsResult, error)`

```go
result, err := bot.Chats.List(ctx, kappelas.GetChatsParams{Limit: 20, Offset: 0})
fmt.Println(result.Chats, result.HasMore)
```

#### `Chats.Iterate(ctx, pageSize, fn)` → `error`

```go
err := bot.Chats.Iterate(ctx, 50, func(chat *kappelas.Chat) bool {
    fmt.Println(chat.ChatID, chat.Type)
    return true // return false to stop early
})
```

---

### `webhooks`

#### `Webhooks.Set(ctx, params)` → `(*WebhookSetResult, error)`

```go
bot.Webhooks.Set(ctx, kappelas.SetWebhookParams{
    URL: "https://your-server.com/kappela-webhook",
})
```

#### `Webhooks.GetInfo(ctx)` → `(*WebhookInfo, error)`

```go
info, err := bot.Webhooks.GetInfo(ctx)
// → WebhookInfo{Active: true, URL: &"https://...", CreatedAt: &1234567890}
```

#### `Webhooks.Delete(ctx)` → `(*WebhookDeleteResult, error)`

```go
bot.Webhooks.Delete(ctx)
// → WebhookDeleteResult{Active: false}
```

---

### `profile`

#### `Profile.Get(ctx)` → `(*BotProfile, error)` / `(*UserProfile, error)`

```go
// Bot
profile, err := bot.Profile.Get(ctx)
// → BotProfile{UserID, Username, IsBot: true, About, Description, AvatarURL}

// User
profile, err := me.Profile.Get(ctx)
// → UserProfile{ID, Username, Nom, IsBot: false, IsPremium, AvatarURL, ...}
```

---

## Keyboards

Three types of keyboard can be passed as `ReplyMarkup` on any `Send*` call:

```go
// Inline buttons — attached to the message
yes, no := "yes", "no"
inline := kappelas.InlineKeyboard{
    InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
        {Text: "Yes", CallbackData: &yes},
        {Text: "No",  CallbackData: &no},
    }},
}

// Reply keyboard — shown below the input bar
reply := kappelas.ReplyKeyboard{
    Keyboard: [][]string{
        {"Option A", "Option B"},
        {"Cancel"},
    },
}

// Scroll keyboard — horizontal scrollable chips
scroll := kappelas.ScrollKeyboard{
    ScrollKeyboard: []string{"Small", "Medium", "Large"},
}

bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID:      42,
    Text:        "Pick one:",
    ReplyMarkup: inline,
})
```

---

## Error handling

All API errors return a `*KappelaError` with structured fields:

```go
import "errors"

_, err := bot.Messages.Send(ctx, kappelas.SendMessageParams{ChatID: 999, Text: "Hi"})
if err != nil {
    var e *kappelas.KappelaError
    if errors.As(err, &e) {
        e.Code      // kappelas.ErrCodeNotFound
        e.Status    // 404
        e.Message   // server error message
        e.RequestID // mention this when contacting support
        fmt.Println(e) // full formatted block with hints and solutions
    }
}
```

### Error codes

| Code | HTTP | Meaning |
|------|------|---------|
| `ErrCodeUnauthorized` | 401 | Token or API key invalid / expired |
| `ErrCodeForbidden` | 403 | Missing permission or role |
| `ErrCodeNotFound` | 404 | Resource does not exist |
| `ErrCodeMissingField` | 400 | Required parameter missing |
| `ErrCodeInvalidField` | 400 | Parameter has wrong type or format |
| `ErrCodeConflict` | 409 | Resource already exists |
| `ErrCodeMethodNotAllowed` | 405 | Wrong HTTP method |
| `ErrCodeInvalidPath` | 404 | API path does not exist |
| `ErrCodeInternalError` | 500 | Unexpected server error |
| `ErrCodeServiceUnavailable` | 503 | Service temporarily down |
| `ErrCodeUpstreamError` | 502 | Upstream service error |

---

## File input

Media methods accept a `FileInput` struct:

```go
type FileInput struct {
    Data        []byte
    Filename    string
    ContentType string
}
```

```go
// From disk
data, _ := os.ReadFile("photo.jpg")
bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
    ChatID: 42,
    File:   kappelas.FileInput{Data: data, Filename: "photo.jpg", ContentType: "image/jpeg"},
})

// From memory
bot.Messages.SendDocument(ctx, kappelas.SendMediaParams{
    ChatID: 42,
    File:   kappelas.FileInput{Data: pdfBytes, Filename: "report.pdf", ContentType: "application/pdf"},
})
```

---

## License

MIT © Arnel LAWSON

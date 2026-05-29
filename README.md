# kappelas-sdk-go

[![Go Reference](https://pkg.go.dev/badge/github.com/Arnel7/kappelas-sdk-go.svg)](https://pkg.go.dev/github.com/Arnel7/kappelas-sdk-go)
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
  - [WebSocket (development)](#websocket-development)
  - [Webhook (production)](#webhook-production)
  - [Event reference](#event-reference)
  - [`bot.Reply()` — convenience shorthand](#botreply--convenience-shorthand)
  - [`Message` fields](#message-fields)
  - [`CallbackQuery` fields](#callbackquery-fields)
  - [⚠️ `SenderName` vs `SenderNom`](#️-sendername-vs-sendernom)
- [API reference](#api-reference)
  - [Constructor options](#constructor-options)
  - [messages](#messages)
  - [chats](#chats)
  - [Groups \& channels](#groups--channels)
  - [Chat member management](#chat-member-management)
  - [Invite links](#invite-links)
  - [webhooks](#webhooks)
  - [profile](#profile)
- [Keyboards](#keyboards)
- [Text formatting](#text-formatting)
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
go get github.com/Arnel7/kappelas-sdk-go
```

Requires **Go 1.21+**.

---

## Quick start

### Bot

```go
package main

import (
    "context"

    "github.com/Arnel7/kappelas-sdk-go"
)

func main() {
    ctx := context.Background()
    bot := kappelas.NewBot("YOUR_BOT_TOKEN")

    bot.OnMessage(func(msg *kappelas.Message) {
        bot.Reply(ctx, msg, "Echo: "+*msg.Text)
    })

    bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
        bot.Reply(ctx, cb, "Tu as cliqué : "+cb.CallbackData)
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

bot.OnMessage(func(msg *kappelas.Message) { /* … */ })
bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) { /* … */ })

bot.Start()   // non-blocking, auto-reconnects on disconnect
select {}
```

### Webhook (production)

```go
import (
    "io"
    "net/http"

    "github.com/Arnel7/kappelas-sdk-go"
)

bot := kappelas.NewBot("YOUR_BOT_TOKEN")

// register once
ctx := context.Background()
bot.Webhooks.Set(ctx, kappelas.SetWebhookParams{
    URL: "https://your-server.com/kappela-webhook",
})

bot.OnMessage(func(msg *kappelas.Message) { /* … */ })
bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) { /* … */ })

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

> **WebSocket reconnection** — by default the SDK retries up to 12 times with exponential back-off. Override with `WithWSMaxRetries(n)`. When retries are exhausted `OnDisconnected` fires with code `1006` and reason `"max retries reached"`.

---

### `bot.Reply()` — convenience shorthand

`Reply` sends a text reply without having to repeat `ChatID` and `ReplyToID` manually.

- Called with a **`*Message`** — sets `ReplyToID` automatically (shows a quote banner in the chat).
- Called with a **`*CallbackQuery`** — sends to the same chat, no quote banner (callback queries have no message ID).

```go
bot.OnMessage(func(msg *kappelas.Message) {
    ctx := context.Background()

    // Simple reply
    bot.Reply(ctx, msg, "Reçu 👍")

    // With an inline keyboard
    bot.Reply(ctx, msg, "Choisis une option :", kappelas.SendMessageParams{
        ReplyMarkup: kappelas.InlineKeyboard{
            InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
                {Text: "✅ Oui", CallbackData: ptr("yes")},
                {Text: "❌ Non", CallbackData: ptr("no")},
            }},
        },
    })
})

bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
    ctx := context.Background()
    bot.Reply(ctx, cb, "Tu as cliqué : "+cb.CallbackData)
})
```

`ChatID`, `ReplyToID`, and `Text` in the optional `SendMessageParams` are filled automatically — you only need to set extra fields like `ReplyMarkup` or `DeletePrevious`.

---

### `Message` fields

```go
bot.OnMessage(func(msg *kappelas.Message) {
    msg.ID               // int64         — unique message ID
    msg.ChatID           // int64         — conversation ID
    msg.ChatType         // *ChatType     — "private" | "group" | "channel" (may be nil for history)
    msg.SenderID         // *string       — UUID of the sender (nil for system messages)
    msg.Type             // MessageType   — "text" | "image" | "video" | "audio" | "document" | …
    msg.Text             // *string       — text content (nil for media-only messages)
    msg.MediaID          // *string       — server-side media ID
    msg.ExtraData        // json.RawMessage — inline keyboard payload (when attached)
    msg.Status           // MessageStatus — "sent" | "delivered" | "read"
    msg.CreatedAt        // int64         — Unix timestamp (seconds)
    msg.EditedAt         // *int64        — last edit time, or nil
    msg.DeletedAt        // *int64        — deletion time, or nil
    msg.ReplyToID        // *int64        — ID of the message being replied to
    msg.ReplyToSnapshot  // *ReplySnapshot — snapshot of the replied-to message
    msg.Mentions         // []string      — UUIDs of mentioned users
    msg.SenderName       // *string       — display name in groups/channels (nil in private)
    msg.SenderAvatarURL  // *string       — avatar URL of the sender
    msg.ExpiresAt        // *int64        — expiry time for ephemeral messages
})
```

**`MessageType` values**

| Value | Description |
|-------|-------------|
| `"text"` | Plain text message |
| `"image"` | Photo |
| `"video"` | Video |
| `"audio"` | Audio file |
| `"document"` | Generic file |
| `"sticker"` | Sticker |
| `"poll"` | Poll |
| `"location"` | Location pin |
| `"contact"` | Contact card |
| `"system"` | System notification (member joined, etc.) |

---

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

### ⚠️ `SenderName` vs `SenderNom`

These two fields look similar but come from different event types:

| Field | Event type | Notes |
|-------|-----------|-------|
| `msg.SenderName` | `*Message` | Display name in groups/channels. **Nil in private chats.** |
| `cb.SenderNom` | `*CallbackQuery` | Display name of the user who clicked the button. |

Copy-pasting a handler from `OnMessage` to `OnCallbackQuery` (or vice-versa) gives a compile error — which is intentional.

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
replyTo := int64(123)

result, err := bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID:    42,
    Text:      "Bonjour !",
    ReplyToID: &replyTo,          // optional — shows a quote banner
    ReplyMarkup: kappelas.InlineKeyboard{
        InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
            {Text: "Oui", CallbackData: &yes},
            {Text: "Non", CallbackData: &no},
        }},
    },
})
// → &SendResult{MessageID: …, CreatedAt: …}
```

Pass `DeletePrevious: true` to automatically remove the previous message from this bot in the same chat before sending.

#### `Messages.SendPhoto(ctx, params)` → `(*SendMediaResult, error)`

```go
data, _ := os.ReadFile("banner.png")
result, err := bot.Messages.SendPhoto(ctx, kappelas.SendMediaParams{
    ChatID:  42,
    File:    kappelas.FileInput{Data: data, Filename: "banner.png", ContentType: "image/png"},
    Caption: "Voici notre bannière !",
})
// → SendMediaResult{MessageID, CreatedAt, MediaID}
```

#### `Messages.SendVideo` / `SendDocument` / `SendAudio` → `(*SendMediaResult, error)`

Same shape — pass the appropriate `FileInput`.

#### `Messages.SendCarousel(ctx, params)` → `(*SendCarouselResult, error)`

```go
btn := "Voir"
bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
    ChatID: 42,
    Text:   "Choisissez un produit :",
    Carousel: []kappelas.CarouselCard{
        {ID: "p1", Title: "Produit A", Subtitle: ptr("9 900 FCFA"), ButtonText: &btn},
        {ID: "p2", Title: "Produit B", Subtitle: ptr("19 900 FCFA"), ButtonText: &btn},
    },
    QuickReplyButtons: []kappelas.ScrollKeyboardButton{
        {Text: "Voir plus"}, {Text: "Annuler"},
    },
})
```

When a user clicks a carousel card button, a `CallbackQuery` fires with `CallbackData` set to the card's `ID` (`"p1"`, `"p2"`, …).

#### `Messages.Edit(ctx, params)` → `(*EditMessageResult, error)`

```go
// Edit text only
bot.Messages.Edit(ctx, kappelas.EditMessageParams{
    ChatID: 42, MessageID: 123, NewText: "Mis à jour !",
})

// Edit inline keyboard only (keep existing text)
import "encoding/json"
done := "done"
kb, _ := json.Marshal(kappelas.InlineKeyboard{
    InlineKeyboard: [][]kappelas.InlineKeyboardButton{
        {{Text: "Terminé ✅", CallbackData: &done}},
    },
})
bot.Messages.Edit(ctx, kappelas.EditMessageParams{
    ChatID: 42, MessageID: 123, NewExtraData: kb,
})

// Remove the keyboard entirely
bot.Messages.Edit(ctx, kappelas.EditMessageParams{
    ChatID: 42, MessageID: 123, NewExtraData: json.RawMessage("null"),
})
// → EditMessageResult{Edited: true, MessageID: 123}
```

#### `Messages.SendTyping(ctx, params)` → `(*TypingResult, error)`

```go
bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: 42})        // show
f := false
bot.Messages.SendTyping(ctx, kappelas.SendTypingParams{ChatID: 42, IsTyping: &f}) // hide
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

**`Chat` fields**

```go
chat.ChatID              // int64         — conversation ID
chat.Type                // ChatType      — "private" | "group" | "channel"
chat.Title               // *string       — group/channel name (nil for private)
chat.Participants        // []Participant — members (private only; empty for large groups)
chat.LastMessageAt       // *string       — ISO 8601 timestamp of last message
chat.CreatedAt           // string        — ISO 8601 creation timestamp
chat.IsPublic            // bool          — public group or channel
chat.OnlyAdminsCanWrite  // bool          — only admins can post
chat.Description         // *string       — group/channel description
chat.AvatarURL           // *string       — avatar image URL
```

---

### Groups & channels

#### `Chats.GetMyGroups(ctx)` → `(*GetMyGroupsResult, error)`

Returns every group and channel the bot belongs to, with the bot's role in each.

```go
result, err := bot.Chats.GetMyGroups(ctx)
for _, g := range result.Groups {
    title := "(sans titre)"
    if g.Title != nil {
        title = *g.Title
    }
    fmt.Printf("%d (%s) %q → %s\n", g.ChatID, g.Type, title, g.BotRole)
}

// Filter to groups where the bot is admin
for _, g := range result.Groups {
    if g.BotRole == kappelas.ParticipantRoleAdmin {
        // can create invite links, manage members…
    }
}
```

`BotGroupEntry` fields:

| Field | Type | Description |
|-------|------|-------------|
| `ChatID` | `int64` | Conversation ID |
| `Type` | `ChatType` | `"group"` or `"channel"` (never `"private"`) |
| `Title` | `*string` | Group or channel name |
| `ParticipantCount` | `int` | Total members (including the bot) |
| `BotRole` | `ParticipantRole` | `"member"` or `"admin"` |

---

### Chat member management

All methods below require the bot to be **a member** of the conversation.  
Methods that modify membership (`AddMember`, `BanMember`, `PromoteMember`) additionally require **admin rights**.

#### `Chats.GetAdministrators(ctx, params)` → `(*GetChatAdministratorsResult, error)`

```go
result, err := bot.Chats.GetAdministrators(ctx, kappelas.GetChatAdministratorsParams{
    ChatID: 42,
})
for _, admin := range result.Admins {
    fmt.Println(admin.UserID, admin.Role) // role is always "admin"
}
```

#### `Chats.GetMember(ctx, params)` → `(*ChatMemberInfo, error)`

Returns the role of a specific member. Returns `ErrCodeNotFound` if the user is not in the conversation.

```go
member, err := bot.Chats.GetMember(ctx, kappelas.GetChatMemberParams{
    ChatID: 42,
    UserID: "user-uuid",
})
fmt.Println(member.Role) // "admin" | "member"
```

#### `Chats.AddMember(ctx, params)` → `(*AddChatMemberResult, error)`

```go
bot.Chats.AddMember(ctx, kappelas.AddChatMemberParams{
    ChatID: 42,
    UserID: "user-uuid",
})
```

#### `Chats.BanMember(ctx, params)` → `(*BanChatMemberResult, error)`

Removes (kicks) a user. To remove the bot itself, use `LeaveChat` instead.

```go
bot.Chats.BanMember(ctx, kappelas.BanChatMemberParams{
    ChatID: 42,
    UserID: "user-uuid",
})
```

#### `Chats.PromoteMember(ctx, params)` → `(*PromoteChatMemberResult, error)`

```go
// Promote to admin
bot.Chats.PromoteMember(ctx, kappelas.PromoteChatMemberParams{
    ChatID: 42,
    UserID: "user-uuid",
    Role:   kappelas.ParticipantRoleAdmin,
})

// Demote back to member
bot.Chats.PromoteMember(ctx, kappelas.PromoteChatMemberParams{
    ChatID: 42,
    UserID: "user-uuid",
    Role:   kappelas.ParticipantRoleMember,
})
```

#### `Chats.LeaveChat(ctx, params)` → `(*LeaveChatResult, error)`

```go
bot.Chats.LeaveChat(ctx, kappelas.LeaveChatParams{ChatID: 42})
```

---

### Invite links

All invite link methods require **admin rights**.

#### `Chats.CreateInviteLink(ctx, params)` → `(*ChatInviteLink, error)`

```go
// Permanent link, unlimited uses
link, err := bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
    ChatID: 42,
})
fmt.Println(link.URL) // "https://kappelas.com/invite/aBcD123xyz"

// Single-use, expires in 24 h
link, err := bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
    ChatID:    42,
    MaxUses:   1,
    ExpiresIn: "24h",
})
```

`ExpiresIn` values: `"1h"` · `"24h"` · `"7d"` · `"30d"` · `"never"` (default)

#### `Chats.CreateSingleUseInviteLink(ctx, params)` → `(*ChatInviteLink, error)`

Shorthand for `CreateInviteLink` with `MaxUses: 1`.

```go
link, err := bot.Chats.CreateSingleUseInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
    ChatID: 42,
})
```

#### `Chats.GetInviteLinks(ctx, params)` → `(*GetChatInviteLinksResult, error)`

```go
result, err := bot.Chats.GetInviteLinks(ctx, kappelas.GetChatInviteLinksParams{ChatID: 42})
for _, link := range result.InviteLinks {
    max := "∞"
    if link.MaxUses > 0 {
        max = strconv.Itoa(link.MaxUses)
    }
    fmt.Printf("%s — %d/%s uses\n", link.URL, link.UseCount, max)
}
```

#### `Chats.RevokeInviteLink(ctx, params)` → `(*RevokeChatInviteLinkResult, error)`

```go
result, err := bot.Chats.RevokeInviteLink(ctx, kappelas.RevokeChatInviteLinkParams{
    ChatID: 42,
    Code:   "aBcD123xyz", // link.Code from CreateInviteLink
})
fmt.Println(result.Revoked) // true
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
// → WebhookInfo{Active: true, URL: &"https://…", CreatedAt: &1234567890}
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
// → UserProfile{ID, Username, Nom, IsBot: false, IsPremium, AvatarURL, …}
```

---

## Keyboards

Three keyboard types can be passed as `ReplyMarkup` on any `Send*` call.

### Inline keyboard — attached to the message

```go
yes, no := "yes", "no"
inline := kappelas.InlineKeyboard{
    InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
        {Text: "✅ Oui", CallbackData: &yes},
        {Text: "❌ Non", CallbackData: &no},
    }},
}
```

### Reply keyboard — shown below the input bar

Buttons trigger a `CallbackQuery` when tapped.

**Short form** — label and callback value are the same:

```go
reply := kappelas.ReplyKeyboard{
    Keyboard: [][]kappelas.ReplyKeyboardButton{
        {{Text: "Option A"}, {Text: "Option B"}},
        {{Text: "Annuler"}},
    },
}
```

**Long form** — separate label and callback value:

```go
reply := kappelas.ReplyKeyboard{
    Keyboard: [][]kappelas.ReplyKeyboardButton{
        {
            {Text: "✅ Confirmer", CallbackData: "confirm_yes"},
            {Text: "❌ Annuler",   CallbackData: "confirm_no"},
        },
    },
}
```

### Scroll keyboard — horizontal scrollable chips

```go
// Short form
scroll := kappelas.ScrollKeyboard{
    ScrollKeyboard: []kappelas.ScrollKeyboardButton{
        {Text: "Petit"}, {Text: "Moyen"}, {Text: "Grand"},
    },
}

// Long form — emoji label, clean callback value
scroll := kappelas.ScrollKeyboard{
    ScrollKeyboard: []kappelas.ScrollKeyboardButton{
        {Text: "📦 Commandes", CallbackData: "menu_orders"},
        {Text: "❓ Aide",      CallbackData: "menu_help"},
    },
}
```

```go
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID:      42,
    Text:        "Choisis une option :",
    ReplyMarkup: inline, // or reply, or scroll
})
```

---

## Text formatting

Kappela renders a subset of Markdown inside message text.

| Syntax | Result |
|--------|--------|
| `**bold**` | **bold** |
| `_italic_` | *italic* |
| `` `code` `` | inline monospace |
| ` ```block``` ` | full-width code block with copy button |
| `[label](https://url)` | clickable link |

**Automatic detection** — the platform auto-links the following without explicit Markdown:

- **URLs** — domains ending in `com org net fr io dev co me app tech info biz xyz eu uk de ru tv cc gg ai be ch ca` are detected. African TLDs (`.bj`, `.sn`, `.ci`) are **not** auto-detected — wrap them in `[label](url)` syntax.
- **Phone numbers** — `+229 0162861571`, `+33 6 12 34 56 78`, `06 12 34 56 78` (spaces, dashes, parentheses accepted).

> **Block vs inline code** — wrapping code on its own line (` ```…``` `) renders a full card with a copy button. Wrapping inside a sentence (`` `value` ``) renders as styled monospace text inline.

---

## Error handling

All API errors return a `*KappelaError` with structured fields:

```go
import "errors"

_, err := bot.Messages.Send(ctx, kappelas.SendMessageParams{ChatID: 999, Text: "Hi"})
if err != nil {
    var e *kappelas.KappelaError
    if errors.As(err, &e) {
        e.Code      // kappelas.ErrCodeForbidden
        e.Status    // 403
        e.Message   // server error message
        e.RequestID // mention this when contacting support
        fmt.Println(e) // formatted block with hints
    }
}
```

### Error codes

| Code | HTTP | Meaning |
|------|------|---------|
| `ErrCodeUnauthorized` | 401 | Token or API key invalid / expired |
| `ErrCodeForbidden` | 403 | Missing permission or role (bot not in chat, not admin…) |
| `ErrCodeNotFound` | 404 | Resource does not exist |
| `ErrCodeMissingField` | 400 | Required parameter missing |
| `ErrCodeInvalidField` | 400 | Parameter has wrong type or format |
| `ErrCodeConflict` | 409 | Resource already exists |
| `ErrCodeMethodNotAllowed` | 405 | Wrong HTTP method |
| `ErrCodeInvalidPath` | 404 | API path does not exist |
| `ErrCodeInternalError` | 500 | Unexpected server error |
| `ErrCodeServiceUnavailable` | 503 | Service temporarily down |
| `ErrCodeUpstreamError` | 502 | Upstream service error |

> `ErrCodeForbidden` (not `ErrCodeNotFound`) is returned when the bot tries to send a message to a chat it has never joined.

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
    File:   kappelas.FileInput{Data: pdfBytes, Filename: "rapport.pdf", ContentType: "application/pdf"},
})
```

> The Go SDK accepts raw bytes only. To send a file from an HTTPS URL, fetch it first with `http.Get` and pass the response body as `Data`.

---

## License

MIT © Arnel LAWSON

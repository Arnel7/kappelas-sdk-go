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
- [API reference](#api-reference)
  - [Constructor options](#constructor-options)
  - [messages](#messages)
  - [chats](#chats)
  - [Groups \& channels](#groups--channels)
  - [Receiving group messages](#receiving-group-messages)
  - [Replying in a group](#replying-in-a-group)
  - [Getting member IDs](#getting-member-ids)
  - [Detecting conversation type](#detecting-conversation-type)
  - [Full group bot example](#full-group-bot-example)
  - [Chat member management](#chat-member-management)
  - [Invite links](#invite-links)
  - [webhooks](#webhooks)
  - [profile](#profile)
- [Keyboards](#keyboards)
  - [Comparison](#comparison)
  - [Inline keyboard](#inline-keyboard--attached-to-the-message)
  - [Reply keyboard](#reply-keyboard--shown-below-the-input-bar)
  - [Scroll keyboard](#scroll-keyboard--horizontal-scrollable-chips)
  - [Full example — all three in one bot](#full-example--all-three-in-one-bot)
- [Text formatting](#text-formatting)
  - [Inline styles](#inline-styles)
  - [Block code](#block-code)
  - [Blockquote / citation](#blockquote--citation)
  - [Mentions and commands](#mentions-and-commands)
  - [Auto-detected links](#auto-detected-links)
  - [Combining formats](#combining-formats)
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
        bot.Reply(ctx, msg, "Echo: "+msg.GetText())
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
    fmt.Printf("[%d] %s\n", msg.ChatID, msg.GetText())
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
    msg.Text             // *string       — text content (nil for media-only messages); use msg.GetText() to avoid nil deref
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
    cb.SenderName      // *string — display name (e.g. "Arnel LAWSON")  use cb.GetSenderName() to avoid nil deref
    cb.SenderUsername  // *string — username (e.g. "arnell")
    cb.CallbackData    // string  — value set on the button
    cb.SentAt          // int64   — Unix timestamp (seconds)
})
```

> Clicks are deduplicated server-side — your handler fires exactly once per click.

---

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

Bots work identically in private chats, groups, and channels — same API, same events. The only requirement is that **the bot must be a member** of the conversation.

#### Receiving group messages

When a bot is added to a group or channel, it automatically receives every message posted there via the same `OnMessage` handler used for DMs.

```go
bot.OnMessage(func(msg *kappelas.Message) {
    // msg.ChatID    — the group's id
    // msg.ChatType  — pointer to "private" | "group" | "channel"
    // msg.SenderID  — UUID of the user who sent the message
    // msg.Text      — message content (nil for media-only)
})
```

> The `ChatType` field lets you distinguish where a message came from without an extra API call.

#### Replying in a group

`ReplyToID` attaches a quote banner to your message. It works identically in private chats, groups, and channels. In groups, always quote the user you're responding to — it makes the context clear to all members.

```go
bot.OnMessage(func(msg *kappelas.Message) {
    ctx := context.Background()
    name := msg.GetSenderName() // "" in private chats, display name in groups/channels
    if name == "" {
        name = "ami"
    }
    bot.Messages.Send(ctx, kappelas.SendMessageParams{
        ChatID:    msg.ChatID,
        Text:      "Reçu, " + name + " 👋",
        ReplyToID: &msg.ID, // quotes the original message
    })
})
```

**Quoting any historical message** — `ReplyToID` accepts any `MessageID`, not just the one that triggered the event:

```go
historyID := int64(456)
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID:    msg.ChatID,
    Text:      "Voici la réponse à ta question précédente :",
    ReplyToID: &historyID,
})
```

**Works on all Send* methods** — `SendPhoto`, `SendVideo`, `SendDocument`, `SendAudio`, and `SendCarousel` all accept `ReplyToID`:

```go
replyTo := msg.ID
bot.Messages.SendCarousel(ctx, kappelas.SendCarouselParams{
    ChatID:    msg.ChatID,
    Text:      "Voici nos produits :",
    Carousel:  []kappelas.CarouselCard{{ID: "p1", Title: "Produit A"}},
    ReplyToID: &replyTo, // banner shows above the carousel
})
```

#### Getting member IDs

There are three ways to obtain the `UserID` of members in a group or channel:

**1. From incoming messages** — the simplest. `msg.SenderID` is always set on every message event:

```go
bot.OnMessage(func(msg *kappelas.Message) {
    if msg.ChatType != nil && *msg.ChatType == kappelas.ChatTypeGroup {
        fmt.Println(*msg.SenderID)   // UUID of the sender
        fmt.Println(*msg.SenderName) // display name (nil in private chats)
    }
})
```

**2. From the participants list** — `Chats.List()` returns the full member list for each chat:

```go
result, err := bot.Chats.List(ctx, kappelas.GetChatsParams{Limit: 50})
for _, chat := range result.Chats {
    for _, member := range chat.Participants {
        fmt.Println(member.ID)     // UUID — use as UserID in member calls
        fmt.Println(member.Nom)    // display name
        fmt.Println(member.IsBot)  // true if this participant is a bot
        if member.Role != nil {
            fmt.Println(*member.Role) // "admin" | "member" (nil on private chats)
        }
    }
}
```

**3. From `Chats.GetAdministrators()`** — when you only need admin IDs:

```go
result, err := bot.Chats.GetAdministrators(ctx, kappelas.GetChatAdministratorsParams{ChatID: 42})
for _, admin := range result.Admins {
    fmt.Println(admin.UserID) // UUID
}
```

> `Chats.GetMember()` lets you check whether a specific user is still in the group and what their current role is — useful after a `BanMember` or `PromoteMember` call to confirm the change.

#### Detecting conversation type

`msg.ChatType` is available on every incoming message. Use it to adapt bot behaviour per context:

```go
bot.OnMessage(func(msg *kappelas.Message) {
    ctx := context.Background()
    if msg.ChatType == nil {
        return
    }
    switch *msg.ChatType {
    case kappelas.ChatTypePrivate:
        // 1-on-1 chat — show full keyboard, personalise replies
        bot.Messages.Send(ctx, kappelas.SendMessageParams{
            ChatID: msg.ChatID,
            Text:   "De quoi as-tu besoin ?",
            ReplyMarkup: kappelas.ScrollKeyboard{
                ScrollKeyboard: []kappelas.ScrollKeyboardButton{
                    {Text: "📦 Commandes"}, {Text: "❓ Aide"}, {Text: "⚙️ Réglages"},
                },
            },
        })

    case kappelas.ChatTypeGroup:
        // Multi-user — reply with a quote so context is clear
        bot.Messages.Send(ctx, kappelas.SendMessageParams{
            ChatID:    msg.ChatID,
            Text:      "✅ Noté !",
            ReplyToID: &msg.ID,
        })

    case kappelas.ChatTypeChannel:
        // Bot-only posting — no user interaction expected
    }
})
```

#### Full group bot example

A bot that works across private chats, groups, and channels:

```go
package main

import (
    "context"
    "fmt"

    "github.com/Arnel7/kappelas-sdk-go"
)

func main() {
    ctx := context.Background()
    bot := kappelas.NewBot("YOUR_BOT_TOKEN")

    bot.OnMessage(func(msg *kappelas.Message) {
        text := msg.GetText()
        if text == "" {
            return
        }

        isGroup   := msg.ChatType != nil && *msg.ChatType == kappelas.ChatTypeGroup
        isPrivate := msg.ChatType != nil && *msg.ChatType == kappelas.ChatTypePrivate

        // /status command — works anywhere
        if text == "/status" {
            p := kappelas.SendMessageParams{
                ChatID: msg.ChatID,
                Text:   "🟢 Bot en ligne",
            }
            if isGroup {
                p.ReplyToID = &msg.ID // quote in groups
            }
            bot.Messages.Send(ctx, p)
            return
        }

        // /invite command — admin-only, group/channel only
        if text == "/invite" && !isPrivate {
            link, err := bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
                ChatID: msg.ChatID,
            })
            if err != nil {
                bot.Messages.Send(ctx, kappelas.SendMessageParams{
                    ChatID: msg.ChatID,
                    Text:   "❌ J'ai besoin des droits admin pour créer des liens d'invitation.",
                })
                return
            }
            p := kappelas.SendMessageParams{
                ChatID: msg.ChatID,
                Text:   fmt.Sprintf("🔗 Lien d'invitation : %s", link.URL),
            }
            if isGroup {
                p.ReplyToID = &msg.ID
            }
            bot.Messages.Send(ctx, p)
            return
        }

        // Private only — interactive keyboard
        if isPrivate {
            yes, help := "orders", "help"
            bot.Messages.Send(ctx, kappelas.SendMessageParams{
                ChatID: msg.ChatID,
                Text:   "De quoi as-tu besoin ?",
                ReplyMarkup: kappelas.InlineKeyboard{
                    InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
                        {Text: "📦 Commandes", CallbackData: &yes},
                        {Text: "❓ Aide",      CallbackData: &help},
                    }},
                },
            })
        }
    })

    bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
        bot.Messages.Send(ctx, kappelas.SendMessageParams{
            ChatID: cb.ChatID,
            Text:   "Tu as choisi : " + cb.CallbackData,
        })
    })

    bot.Start()
    select {}
}
```

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

### Comparison

| | Inline | Reply | Scroll |
|---|---|---|---|
| Position | Attached to the message | Below the input bar | Horizontal chips above input |
| Stays after tap | ✅ Yes | ❌ Dismissed | ✅ Yes |
| Separate `CallbackData` | ✅ Always | ✅ Yes (long form) | ✅ Yes (long form) |
| URL button | ✅ Yes | ❌ No | ❌ No |
| Layout | 2-D grid `[][]` | 2-D grid `[][]` | 1-D list `[]` |

---

### Inline keyboard — attached to the message

Buttons stay visible after being tapped. Each button fires a `CallbackQuery` (`CallbackData`) or opens a URL (`URL`).

```go
yes, no := "yes", "no"
url := "https://kappelas.com"
inline := kappelas.InlineKeyboard{
    InlineKeyboard: [][]kappelas.InlineKeyboardButton{
        {
            {Text: "✅ Confirmer", CallbackData: &yes},
            {Text: "❌ Annuler",   CallbackData: &no},
        },
        {
            {Text: "🌐 Site web", URL: &url},
        },
    },
}
```

### Reply keyboard — shown below the input bar

Dismissed after the user taps a button. Buttons trigger a `CallbackQuery`.

**Short form** — label and callback value are the same:

```go
reply := kappelas.ReplyKeyboard{
    Keyboard: [][]kappelas.ReplyKeyboardButton{
        {{Text: "📦 Mes commandes"}, {Text: "❓ Aide"}},
        {{Text: "🔙 Retour"}},
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
        {
            {Text: "↩ Retour", CallbackData: "cancel"},
        },
    },
}
```

You can **mix** short and long buttons in the same grid:

```go
reply := kappelas.ReplyKeyboard{
    Keyboard: [][]kappelas.ReplyKeyboardButton{
        {{Text: "✅ Confirmer", CallbackData: "confirm"}, {Text: "❓ Aide"}},
    },
}
```

### Scroll keyboard — horizontal scrollable chips

A single row of chips, always visible above the input bar.

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
        {Text: "⚙️ Réglages",  CallbackData: "menu_settings"},
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

### Full example — all three in one bot

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
        if msg.GetText() != "/start" {
            return
        }
        // Persistent navigation chips
        orders, help := "menu_orders", "menu_help"
        bot.Messages.Send(ctx, kappelas.SendMessageParams{
            ChatID: msg.ChatID,
            Text:   "Bienvenue ! De quoi as-tu besoin ?",
            ReplyMarkup: kappelas.ScrollKeyboard{
                ScrollKeyboard: []kappelas.ScrollKeyboardButton{
                    {Text: "📦 Commandes", CallbackData: orders},
                    {Text: "❓ Aide",      CallbackData: help},
                },
            },
        })
    })

    bot.OnCallbackQuery(func(cb *kappelas.CallbackQuery) {
        switch cb.CallbackData {
        case "menu_orders":
            // Inline confirm/cancel buttons
            confirm, cancel := "order_confirm", "order_cancel"
            bot.Messages.Send(ctx, kappelas.SendMessageParams{
                ChatID: cb.ChatID,
                Text:   "Confirmer ta dernière commande ?",
                ReplyMarkup: kappelas.InlineKeyboard{
                    InlineKeyboard: [][]kappelas.InlineKeyboardButton{{
                        {Text: "✅ Confirmer", CallbackData: &confirm},
                        {Text: "❌ Annuler",   CallbackData: &cancel},
                    }},
                },
            })

        case "menu_help":
            // Reply keyboard for topic selection
            bot.Messages.Send(ctx, kappelas.SendMessageParams{
                ChatID: cb.ChatID,
                Text:   "Quel sujet ?",
                ReplyMarkup: kappelas.ReplyKeyboard{
                    Keyboard: [][]kappelas.ReplyKeyboardButton{
                        {
                            {Text: "💳 Facturation", CallbackData: "help_billing"},
                            {Text: "🚚 Livraison",   CallbackData: "help_delivery"},
                        },
                        {{Text: "↩ Retour au menu", CallbackData: "menu_back"}},
                    },
                },
            })
        }
    })

    bot.Start()
    select {}
}
```

---

## Text formatting

Kappela renders a **WhatsApp/Telegram-style subset of Markdown** in every message bubble — bot messages, group messages, and private chat messages. All formatting is applied client-side by the Android app; you only need to send the correct markup in the `Text` or `Caption` field.

### Inline styles

| Syntax | Result |
|--------|--------|
| `**bold**` or `*bold*` | **Bold** |
| `__italic__` or `_italic_` | *Italic* |
| `~strikethrough~` | ~~Strikethrough~~ |
| `` `inline code` `` | Monospace with a tinted background |

```go
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   "Commande *confirmée* ✅\nTotal : **24 990 FCFA**\nRef : `ORD-2024-001`",
})
```

### Block code

Triple backticks render as a **block code card** — only when placed on their own line.

| Position | Rendu |
|----------|-------|
| `` `code` `` en cours de phrase | Monospace inline avec fond teinté |
| ` ```code``` ` sur sa propre ligne | Carte pleine largeur + bouton **copier** |

```go
// Inline — reste dans le flux de texte
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   "Ta ref est `ORD-2024-001` — garde-la précieusement.",
})

// Block — doit être sur sa propre ligne pour s'afficher en carte
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   "Ta clé API :\n```\nsk_live_abc123xyz\n```",
})
```

> The code card collapses to a single line with an ellipsis if the content is too long. Tapping anywhere on the card copies the content to the clipboard.

### Blockquote / citation

Prefix a line with `>` to render it as a citation banner (a `┃` bar on the left, italic, slightly faded):

```go
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   "> Question originale ici\n\nVoici ta réponse.",
})
```

> You can combine blockquotes with `ReplyToID` — use `ReplyToID` when you want to quote a specific existing message (the app shows a reply banner); use `>` when you want to render a quote inline within the text itself.

### Mentions and commands

`@username` and `/command` are auto-detected and rendered as tappable blue links:

```go
// Mention a user by their username
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   "Merci @arnell, ta commande est prête !",
})

// Send a command hint
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   "Tape /help pour voir toutes les commandes disponibles.",
})
```

> **Protection rule:** `@` and `/` inside URLs are never formatted. `@buy_something_bot` is treated as a mention, not as `buy` + `_something_bot` (italic).

### Auto-detected links

The renderer automatically makes the following clickable without any markup:

| Pattern | Behaviour |
|---------|-----------|
| `https://…` or `http://…` | Opens in the in-app browser |
| `domain.com`, `domain.io`, `domain.fr` … | Prefixed with `https://` and opened |
| `email@example.com` | Opens the mail app |
| `+229 01 62 86 15 71`, `(229) 0162-861571` | Opens the dialler |

```go
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   "Visitez kappelas.com ou contactez-nous à support@kappelas.com",
})
```

> **Supported domain extensions** — only the following TLDs are auto-linked:
> `com` `org` `net` `fr` `io` `dev` `co` `me` `app` `tech` `info` `biz` `xyz` `eu` `uk` `de` `ru` `tv` `cc` `gg` `ai` `be` `ch` `ca`
>
> Country codes like `.bj`, `.sn`, `.ci` are **not** auto-detected — use a full `https://` URL instead: `https://kappelas.bj`.

> **Phone format** — any sequence of 8+ digits is detected, with spaces, dashes, and parentheses allowed: `+229 01 62 86 15 71`, `+22901628​61571`, `(229) 0162-861571` all open the dialler.

### Combining formats

All inline styles can be combined freely:

```go
lines := []string{
    "🛒 *Récapitulatif de commande*",
    "",
    "> Widget A × 2",
    "",
    "Total : **49 980 FCFA**",
    "Statut : `CONFIRMÉ`",
    "",
    "Des questions ? Contactez support@kappelas.com ou tapez /help",
}
bot.Messages.Send(ctx, kappelas.SendMessageParams{
    ChatID: 42,
    Text:   strings.Join(lines, "\n"),
})
```

Renders as:

```
🛒 Récapitulatif de commande   ← bold

┃ Widget A × 2                 ← blockquote (italic, faded)

Total : 49 980 FCFA            ← bold amount
Statut : CONFIRMÉ              ← monospace badge

Des questions ? Contactez support@kappelas.com ou tapez /help
                               ← email and /help are tappable
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

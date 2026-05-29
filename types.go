package kappelas

import "encoding/json"

// ─── Error codes ──────────────────────────────────────────────────────────────

// ErrorCode is the machine-readable error code returned by the Kappela API.
type ErrorCode string

const (
	ErrCodeUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden          ErrorCode = "FORBIDDEN"
	ErrCodeNotFound           ErrorCode = "NOT_FOUND"
	ErrCodeInvalidField       ErrorCode = "INVALID_FIELD"
	ErrCodeMissingField       ErrorCode = "MISSING_FIELD"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrCodeConflict           ErrorCode = "CONFLICT"
	ErrCodeMethodNotAllowed   ErrorCode = "METHOD_NOT_ALLOWED"
	ErrCodeInvalidPath        ErrorCode = "INVALID_PATH"
	ErrCodeUpstreamError      ErrorCode = "UPSTREAM_ERROR"
)

// ─── Message ─────────────────────────────────────────────────────────────────

// MessageType is the content type of a message.
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeVideo    MessageType = "video"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeDocument MessageType = "document"
	MessageTypeSystem   MessageType = "system"
	MessageTypePoll     MessageType = "poll"
	MessageTypeSticker  MessageType = "sticker"
	MessageTypeLocation MessageType = "location"
	MessageTypeContact  MessageType = "contact"
)

// MessageStatus is the delivery status of a message.
type MessageStatus string

const (
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
)

// ReplySnapshot is a lightweight snapshot of the message being replied to.
type ReplySnapshot struct {
	MessageID int64       `json:"message_id"`
	SenderID  *string     `json:"sender_id"`
	Type      MessageType `json:"type"`
	Text      *string     `json:"text"`
	MediaID   *string     `json:"media_id"`
}

// Message represents a Kappela chat message.
type Message struct {
	ID        int64   `json:"id"`
	ChatID    int64   `json:"chat_id"`
	// ChatType is the type of conversation ("private", "group", "channel").
	// Always present on WS and webhook events; may be absent on history API results.
	ChatType        *ChatType       `json:"chat_type,omitempty"`
	SenderID        *string         `json:"sender_id"`
	Type            MessageType     `json:"type"`
	Text            *string         `json:"text"`
	MediaID         *string         `json:"media_id"`
	ExtraData       json.RawMessage `json:"extra_data"`
	Status          MessageStatus   `json:"status"`
	EditedAt        *int64          `json:"edited_at"`
	DeletedAt       *int64          `json:"deleted_at"`
	CreatedAt       int64           `json:"created_at"`
	ReplyToID       *int64          `json:"reply_to_id"`
	ReplyToSnapshot *ReplySnapshot  `json:"reply_to_snapshot"`
	Mentions        []string        `json:"mentions"`
	ForwardedFrom   json.RawMessage `json:"forwarded_from"`
	ExpiresAt       *int64          `json:"expires_at"`
	// SenderName is the display name of the sender.
	// Only present on messages in groups and channels — absent in private chats.
	// Note: CallbackQuery has SenderNom (not SenderName) — do not confuse the two.
	SenderName      *string `json:"sender_name,omitempty"`
	SenderAvatarURL *string `json:"sender_avatar_url,omitempty"`
	ClientMsgID     string  `json:"client_msg_id,omitempty"`
	Width           *int    `json:"width,omitempty"`
	Height          *int    `json:"height,omitempty"`
}

// ─── Chat ────────────────────────────────────────────────────────────────────

// ChatType is the type of a conversation.
type ChatType string

const (
	ChatTypePrivate ChatType = "private"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)

// ParticipantRole is the role of a member in a group or channel.
type ParticipantRole string

const (
	ParticipantRoleMember ParticipantRole = "member"
	ParticipantRoleAdmin  ParticipantRole = "admin"
)

// Participant is a member of a chat.
type Participant struct {
	ID        string  `json:"id"`
	Nom       string  `json:"nom"`
	IsBot     bool    `json:"is_bot"`
	IsPremium bool    `json:"is_premium"`
	AvatarURL *string `json:"avatar_url"`
	// Role is the member's role in the conversation.
	// Present on groups and channels; absent on private chats.
	Role *ParticipantRole `json:"role,omitempty"`
}

// Chat represents a Kappela conversation.
type Chat struct {
	ChatID             int64         `json:"chat_id"`
	ID                 int64         `json:"id"`
	Type               ChatType      `json:"type"`
	Title              *string       `json:"title"`
	Participants       []Participant `json:"participants"`
	LastMessageAt      *string       `json:"last_message_at"`
	CreatedAt          string        `json:"created_at"`
	CreatedBy          string        `json:"created_by"`
	IsPinned           bool          `json:"is_pinned"`
	IsPremium          bool          `json:"is_premium"`
	IsPublic           bool          `json:"is_public"`
	OnlyAdminsCanWrite bool          `json:"only_admins_can_write"`
	Labels             []string      `json:"labels"`
	Description        *string       `json:"description"`
	AvatarURL          *string       `json:"avatar_url"`
}

// ─── Profiles ────────────────────────────────────────────────────────────────

// BotProfile is the profile returned for a bot account.
type BotProfile struct {
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	IsBot       bool    `json:"is_bot"`
	About       string  `json:"about"`
	Description string  `json:"description"`
	AvatarURL   *string `json:"avatar_url"`
}

// PrivacySetting is a user privacy configuration value.
type PrivacySetting string

const (
	PrivacyEveryone PrivacySetting = "everyone"
	PrivacyContacts PrivacySetting = "contacts"
	PrivacyNobody   PrivacySetting = "nobody"
)

// UserProfile is the profile returned for a personal account.
type UserProfile struct {
	ID            string         `json:"id"`
	Username      string         `json:"username"`
	Nom           string         `json:"nom"`
	IsBot         bool           `json:"is_bot"`
	IsPremium     bool           `json:"is_premium"`
	AvatarURL     *string        `json:"avatar_url"`
	AllowGroupAdd PrivacySetting `json:"allow_group_add"`
	AllowCalls    PrivacySetting `json:"allow_calls"`
}

// ─── Keyboards ───────────────────────────────────────────────────────────────

// InlineKeyboardButton is a button inside an inline keyboard.
type InlineKeyboardButton struct {
	Text         string  `json:"text"`
	CallbackData *string `json:"callback_data,omitempty"`
	URL          *string `json:"url,omitempty"`
}

// InlineKeyboard renders buttons attached to a message.
type InlineKeyboard struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// ReplyKeyboardButton is a single button in a reply or scroll keyboard.
//
// Short form — label and callback value are identical:
//
//	ReplyKeyboardButton{Text: "Option A"}
//
// Long form — different label and callback value:
//
//	ReplyKeyboardButton{Text: "✅ Confirmer", CallbackData: "confirm_yes"}
type ReplyKeyboardButton struct {
	// Text is the label shown on the button.
	Text string
	// CallbackData is the value sent to the webhook when the button is pressed.
	// Defaults to Text when empty.
	CallbackData string
}

// MarshalJSON serialises as a plain string when CallbackData is empty or equals Text,
// and as {"text":…,"callback_data":…} when they differ.
func (b ReplyKeyboardButton) MarshalJSON() ([]byte, error) {
	if b.CallbackData == "" || b.CallbackData == b.Text {
		return json.Marshal(b.Text)
	}
	return json.Marshal(struct {
		Text         string `json:"text"`
		CallbackData string `json:"callback_data"`
	}{b.Text, b.CallbackData})
}

// ScrollKeyboardButton is a button in a scroll (horizontal chips) keyboard.
// Same short/long form as ReplyKeyboardButton.
type ScrollKeyboardButton = ReplyKeyboardButton

// ReplyKeyboard renders a custom keyboard shown below the message input field.
type ReplyKeyboard struct {
	Keyboard [][]ReplyKeyboardButton `json:"keyboard"`
}

// ScrollKeyboard renders a horizontally scrollable chip bar below the message input.
type ScrollKeyboard struct {
	ScrollKeyboard []ScrollKeyboardButton `json:"scroll_keyboard"`
}

// ─── Carousel ────────────────────────────────────────────────────────────────

// CarouselCard is a single card inside a carousel message.
type CarouselCard struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Subtitle   *string `json:"subtitle,omitempty"`
	ImageURL   *string `json:"image_url,omitempty"`
	ButtonText *string `json:"button_text,omitempty"`
}

// ─── Webhook ─────────────────────────────────────────────────────────────────

// WebhookInfo describes the current webhook configuration.
type WebhookInfo struct {
	Active    bool    `json:"active"`
	URL       *string `json:"url"`
	CreatedAt *int64  `json:"created_at"`
}

// ─── Callback query ──────────────────────────────────────────────────────────

// CallbackQuery is fired when a user clicks an inline button.
type CallbackQuery struct {
	ChatID         int64   `json:"chat_id"`
	SenderID       string  `json:"sender_id"`
	// SenderNom is the display name of the user who clicked (e.g. "Arnel LAWSON").
	// Note: Message uses SenderName — do not confuse the two.
	SenderNom      *string `json:"sender_nom"`
	SenderUsername *string `json:"sender_username"`
	CallbackData   string  `json:"callback_data"`
	SentAt         int64   `json:"sent_at"`
}

// ─── Results ─────────────────────────────────────────────────────────────────

// SendResult is returned after sending a text message.
type SendResult struct {
	MessageID int64 `json:"message_id"`
	CreatedAt int64 `json:"created_at"`
}

// SendMediaResult is returned after sending a media message.
type SendMediaResult struct {
	MessageID int64  `json:"message_id"`
	CreatedAt int64  `json:"created_at"`
	MediaID   string `json:"media_id"`
}

// SendCarouselResult is returned after sending a carousel.
type SendCarouselResult struct {
	MessageID int64  `json:"message_id"`
	CreatedAt int64  `json:"created_at"`
	Type      string `json:"type"`
}

// ChatsResult is the paginated response from the list chats endpoint.
type ChatsResult struct {
	Chats   []Chat `json:"chats"`
	HasMore bool   `json:"has_more"`
}

// TypingResult is returned by the sendTyping endpoint.
type TypingResult struct {
	Typing bool `json:"typing"`
}

// DeleteResult is returned by the deleteMessage endpoint.
type DeleteResult struct {
	Deleted bool `json:"deleted"`
}

// WebhookSetResult is returned after registering a webhook.
type WebhookSetResult struct {
	URL    string `json:"url"`
	Active bool   `json:"active"`
}

// WebhookDeleteResult is returned after removing a webhook.
type WebhookDeleteResult struct {
	Active bool `json:"active"`
}

// EditMessageResult is returned after editing a message.
type EditMessageResult struct {
	Edited    bool  `json:"edited"`
	MessageID int64 `json:"message_id"`
}

// ─── Params ──────────────────────────────────────────────────────────────────

// SendMessageParams holds the parameters for sending a text message.
type SendMessageParams struct {
	ChatID    int64  `json:"chat_id"`
	Text      string `json:"text"`
	// ReplyMarkup accepts InlineKeyboard, ReplyKeyboard, or ScrollKeyboard.
	ReplyMarkup    any    `json:"reply_markup,omitempty"`
	ReplyToID      *int64 `json:"reply_to_id,omitempty"`
	DeletePrevious bool   `json:"delete_previous,omitempty"`
}

// FileInput holds a file to be uploaded.
type FileInput struct {
	Data        []byte
	Filename    string
	ContentType string
}

// SendMediaParams holds the parameters for sending a photo, video, document, or audio.
type SendMediaParams struct {
	ChatID         int64
	File           FileInput
	Caption        string
	ReplyToID      *int64
	DeletePrevious bool
	// ReplyMarkup accepts InlineKeyboard, ReplyKeyboard, or ScrollKeyboard.
	ReplyMarkup any
}

// SendCarouselParams holds the parameters for sending a product carousel.
type SendCarouselParams struct {
	ChatID            int64                  `json:"chat_id"`
	Text              string                 `json:"text,omitempty"`
	Carousel          []CarouselCard         `json:"carousel"`
	// QuickReplyButtons are shown as chips below the carousel.
	// Accepts short form {Text: "label"} or long form {Text: "label", CallbackData: "value"}.
	QuickReplyButtons []ScrollKeyboardButton `json:"quick_reply_buttons,omitempty"`
	ReplyToID         *int64                 `json:"reply_to_id,omitempty"`
	DeletePrevious    bool                   `json:"delete_previous,omitempty"`
}

// SendTypingParams holds the parameters for the typing indicator.
// IsTyping defaults to true when nil (show indicator). Set to false to hide it.
type SendTypingParams struct {
	ChatID   int64 `json:"chat_id"`
	IsTyping *bool `json:"is_typing,omitempty"`
}

// EditMessageParams holds the parameters for editing a message.
type EditMessageParams struct {
	ChatID       int64           `json:"chat_id"`
	MessageID    int64           `json:"message_id"`
	NewText      string          `json:"new_text,omitempty"`
	NewExtraData json.RawMessage `json:"new_extra_data,omitempty"`
}

// DeleteMessageParams holds the parameters for deleting a message.
type DeleteMessageParams struct {
	ChatID    int64 `json:"chat_id"`
	MessageID int64 `json:"message_id"`
}

// SetWebhookParams holds the parameters for registering a webhook.
type SetWebhookParams struct {
	URL    string  `json:"url"`
	Secret *string `json:"secret,omitempty"`
}

// GetChatsParams holds the parameters for listing chats.
type GetChatsParams struct {
	Limit  int
	Offset int
}

// ─── Chat member management ──────────────────────────────────────────────────

// ChatMemberInfo is a minimal member record returned by GetMember and GetAdministrators.
type ChatMemberInfo struct {
	UserID string          `json:"user_id"`
	Role   ParticipantRole `json:"role"`
}

// AddChatMemberParams holds the parameters for adding a user to a group or channel.
// The bot must be admin of the conversation.
type AddChatMemberParams struct {
	ChatID int64  `json:"chat_id"`
	UserID string `json:"user_id"`
}

// AddChatMemberResult is returned after successfully adding a user.
type AddChatMemberResult struct {
	Description string `json:"description"`
}

// BanChatMemberParams holds the parameters for removing (kicking) a user.
// The bot must be admin. To remove itself, use LeaveChatParams instead.
type BanChatMemberParams struct {
	ChatID int64  `json:"chat_id"`
	UserID string `json:"user_id"`
}

// BanChatMemberResult is returned after successfully removing a user.
type BanChatMemberResult struct {
	Description string `json:"description"`
}

// LeaveChatParams holds the parameters for the bot to leave a group or channel.
type LeaveChatParams struct {
	ChatID int64 `json:"chat_id"`
}

// LeaveChatResult is returned after the bot leaves.
type LeaveChatResult struct {
	Description string `json:"description"`
}

// PromoteChatMemberParams holds the parameters for changing a member's role.
// The bot must be admin.
type PromoteChatMemberParams struct {
	ChatID int64           `json:"chat_id"`
	UserID string          `json:"user_id"`
	// Role: ParticipantRoleAdmin promotes, ParticipantRoleMember demotes.
	Role ParticipantRole `json:"role"`
}

// PromoteChatMemberResult is returned after a role change.
type PromoteChatMemberResult struct {
	UserID string          `json:"user_id"`
	Role   ParticipantRole `json:"role"`
}

// GetChatAdministratorsParams holds the parameters for fetching chat admins.
type GetChatAdministratorsParams struct {
	ChatID int64 `json:"chat_id"`
}

// GetChatAdministratorsResult contains all admins of a group or channel.
type GetChatAdministratorsResult struct {
	Admins []ChatMemberInfo `json:"admins"`
}

// GetChatMemberParams holds the parameters for looking up a single member.
type GetChatMemberParams struct {
	ChatID int64  `json:"chat_id"`
	UserID string `json:"user_id"`
}

// ─── Invite links ────────────────────────────────────────────────────────────

// ChatInviteLink describes an active invite link for a group or channel.
type ChatInviteLink struct {
	// Code is the short identifier used in the URL (e.g. "aBcD123xyz").
	Code string `json:"code"`
	// URL is the full invite URL (e.g. "https://kappelas.com/invite/aBcD123xyz").
	URL string `json:"url"`
	// MaxUses is the maximum number of allowed uses (0 = unlimited).
	MaxUses int `json:"max_uses"`
	// UseCount is the current number of uses.
	UseCount int `json:"use_count"`
	// ExpiresAt is the expiry as Unix timestamp (seconds), or nil if permanent.
	ExpiresAt *int64 `json:"expires_at"`
	// CreatedAt is the creation time as Unix timestamp (seconds).
	CreatedAt int64 `json:"created_at"`
}

// CreateChatInviteLinkParams holds the parameters for creating an invite link.
// The bot must be admin of the conversation.
type CreateChatInviteLinkParams struct {
	ChatID int64 `json:"chat_id"`
	// MaxUses is 0 for unlimited, or a positive number to cap uses.
	MaxUses int `json:"max_uses,omitempty"`
	// ExpiresIn controls expiry: "1h", "24h", "7d", "30d", or "never" (default).
	ExpiresIn string `json:"expires_in,omitempty"`
}

// GetChatInviteLinksParams holds the parameters for listing invite links.
type GetChatInviteLinksParams struct {
	ChatID int64 `json:"chat_id"`
}

// GetChatInviteLinksResult contains all active invite links for a group or channel.
type GetChatInviteLinksResult struct {
	InviteLinks []ChatInviteLink `json:"invite_links"`
}

// RevokeChatInviteLinkParams holds the parameters for revoking an invite link.
type RevokeChatInviteLinkParams struct {
	ChatID int64 `json:"chat_id"`
	// Code is the code field of the link to revoke.
	Code string `json:"code"`
}

// RevokeChatInviteLinkResult is returned after revoking a link.
type RevokeChatInviteLinkResult struct {
	Revoked bool   `json:"revoked"`
	Code    string `json:"code"`
}

// ─── Bot group membership ────────────────────────────────────────────────────

// BotGroupEntry is a group or channel the bot belongs to, enriched with its role.
type BotGroupEntry struct {
	// ChatID is the conversation ID — use this as ChatID in all API calls.
	ChatID int64 `json:"chat_id"`
	// Type is "group" or "channel". Never "private".
	Type ChatType `json:"type"`
	// Title is the group or channel name.
	Title *string `json:"title"`
	// ParticipantCount is the total number of members (including the bot).
	ParticipantCount int `json:"participant_count"`
	// BotRole is the bot's role in this conversation.
	BotRole ParticipantRole `json:"bot_role"`
}

// GetMyGroupsResult holds the list of groups and channels the bot belongs to.
type GetMyGroupsResult struct {
	Groups []BotGroupEntry `json:"groups"`
}

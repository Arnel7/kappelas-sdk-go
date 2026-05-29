package kappelas

import (
	"context"
	"fmt"
)

// ChatsResource provides methods to access and manage chats.
type ChatsResource struct {
	http *httpClient
	base string
}

// ─── Chat listing ─────────────────────────────────────────────────────────────

// List returns a paginated list of chats accessible to this bot or user.
func (r *ChatsResource) List(ctx context.Context, params GetChatsParams) (ChatsResult, error) {
	path := r.base + "/getChats"
	sep := "?"
	if params.Limit > 0 {
		path += fmt.Sprintf("%slimit=%d", sep, params.Limit)
		sep = "&"
	}
	if params.Offset > 0 {
		path += fmt.Sprintf("%soffset=%d", sep, params.Offset)
	}
	return httpGet[ChatsResult](ctx, r.http, path)
}

// Iterate calls fn for every chat, handling pagination automatically.
// Return false from fn to stop iteration early.
//
// Example:
//
//	err := bot.Chats.Iterate(ctx, 50, func(chat *kappelas.Chat) bool {
//	    fmt.Println(chat.ChatID, chat.Type)
//	    return true // continue
//	})
func (r *ChatsResource) Iterate(ctx context.Context, pageSize int, fn func(*Chat) bool) error {
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := 0
	for {
		result, err := r.List(ctx, GetChatsParams{Limit: pageSize, Offset: offset})
		if err != nil {
			return err
		}
		for i := range result.Chats {
			if !fn(&result.Chats[i]) {
				return nil
			}
		}
		if !result.HasMore || len(result.Chats) == 0 {
			return nil
		}
		offset += len(result.Chats)
	}
}

// ─── Chat member management ───────────────────────────────────────────────────

// AddMember adds a user to a group or channel.
// The bot must be admin of the conversation.
//
// Example:
//
//	result, err := bot.Chats.AddMember(ctx, kappelas.AddChatMemberParams{
//	    ChatID: 42, UserID: "user-uuid",
//	})
func (r *ChatsResource) AddMember(ctx context.Context, params AddChatMemberParams) (*AddChatMemberResult, error) {
	return httpPost[*AddChatMemberResult](ctx, r.http, r.base+"/addChatMember", params)
}

// BanMember removes (kicks) a user from a group or channel.
// The bot must be admin. To remove itself, use LeaveChat instead.
//
// Example:
//
//	result, err := bot.Chats.BanMember(ctx, kappelas.BanChatMemberParams{
//	    ChatID: 42, UserID: "user-uuid",
//	})
func (r *ChatsResource) BanMember(ctx context.Context, params BanChatMemberParams) (*BanChatMemberResult, error) {
	return httpPost[*BanChatMemberResult](ctx, r.http, r.base+"/banChatMember", params)
}

// LeaveChat makes the bot leave a group or channel.
//
// Example:
//
//	result, err := bot.Chats.LeaveChat(ctx, kappelas.LeaveChatParams{ChatID: 42})
func (r *ChatsResource) LeaveChat(ctx context.Context, params LeaveChatParams) (*LeaveChatResult, error) {
	return httpPost[*LeaveChatResult](ctx, r.http, r.base+"/leaveChat", params)
}

// PromoteMember promotes or demotes a member.
// The bot must be admin.
//   - Role: ParticipantRoleAdmin  — grants admin rights
//   - Role: ParticipantRoleMember — revokes admin rights
//
// Example:
//
//	// Promote to admin
//	bot.Chats.PromoteMember(ctx, kappelas.PromoteChatMemberParams{
//	    ChatID: 42,
//	    UserID: "user-uuid",
//	    Role:   kappelas.ParticipantRoleAdmin,
//	})
func (r *ChatsResource) PromoteMember(ctx context.Context, params PromoteChatMemberParams) (*PromoteChatMemberResult, error) {
	return httpPost[*PromoteChatMemberResult](ctx, r.http, r.base+"/promoteChatMember", params)
}

// GetAdministrators returns all admins of a group or channel.
// The bot must be a member of the conversation.
//
// Example:
//
//	result, err := bot.Chats.GetAdministrators(ctx, kappelas.GetChatAdministratorsParams{ChatID: 42})
//	for _, admin := range result.Admins {
//	    fmt.Println(admin.UserID, admin.Role)
//	}
func (r *ChatsResource) GetAdministrators(ctx context.Context, params GetChatAdministratorsParams) (*GetChatAdministratorsResult, error) {
	return httpPost[*GetChatAdministratorsResult](ctx, r.http, r.base+"/getChatAdministrators", params)
}

// GetMember returns info for a specific member (UserID + Role).
// The bot must be a member of the conversation.
// Returns ErrCodeNotFound if the user is not in the conversation.
//
// Example:
//
//	member, err := bot.Chats.GetMember(ctx, kappelas.GetChatMemberParams{
//	    ChatID: 42, UserID: "user-uuid",
//	})
//	fmt.Println(member.Role) // "admin" | "member"
func (r *ChatsResource) GetMember(ctx context.Context, params GetChatMemberParams) (*ChatMemberInfo, error) {
	return httpPost[*ChatMemberInfo](ctx, r.http, r.base+"/getChatMember", params)
}

// ─── Invite links ─────────────────────────────────────────────────────────────

// CreateInviteLink creates an invite link for a group or channel.
// The bot must be admin of the conversation.
//
// Example:
//
//	// Permanent link, unlimited uses
//	link, err := bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
//	    ChatID: 42,
//	})
//	fmt.Println(link.URL) // "https://kappelas.com/invite/aBcD123xyz"
//
//	// Single-use, expires in 24 h
//	link, err := bot.Chats.CreateInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
//	    ChatID: 42, MaxUses: 1, ExpiresIn: "24h",
//	})
func (r *ChatsResource) CreateInviteLink(ctx context.Context, params CreateChatInviteLinkParams) (*ChatInviteLink, error) {
	return httpPost[*ChatInviteLink](ctx, r.http, r.base+"/createChatInviteLink", params)
}

// CreateSingleUseInviteLink is a shorthand to create a single-use invite link.
// Equivalent to CreateInviteLink with MaxUses: 1. The bot must be admin.
//
// Example:
//
//	link, err := bot.Chats.CreateSingleUseInviteLink(ctx, kappelas.CreateChatInviteLinkParams{
//	    ChatID: 42,
//	})
func (r *ChatsResource) CreateSingleUseInviteLink(ctx context.Context, params CreateChatInviteLinkParams) (*ChatInviteLink, error) {
	params.MaxUses = 1
	return r.CreateInviteLink(ctx, params)
}

// GetInviteLinks returns all active invite links for a group or channel.
// The bot must be admin.
//
// Example:
//
//	result, err := bot.Chats.GetInviteLinks(ctx, kappelas.GetChatInviteLinksParams{ChatID: 42})
//	for _, link := range result.InviteLinks {
//	    fmt.Printf("%s — %d/%d uses\n", link.URL, link.UseCount, link.MaxUses)
//	}
func (r *ChatsResource) GetInviteLinks(ctx context.Context, params GetChatInviteLinksParams) (*GetChatInviteLinksResult, error) {
	return httpPost[*GetChatInviteLinksResult](ctx, r.http, r.base+"/getChatInviteLinks", params)
}

// RevokeInviteLink revokes an active invite link so it can no longer be used.
// The bot must be admin.
//
// Example:
//
//	result, err := bot.Chats.RevokeInviteLink(ctx, kappelas.RevokeChatInviteLinkParams{
//	    ChatID: 42, Code: "aBcD123xyz",
//	})
func (r *ChatsResource) RevokeInviteLink(ctx context.Context, params RevokeChatInviteLinkParams) (*RevokeChatInviteLinkResult, error) {
	return httpPost[*RevokeChatInviteLinkResult](ctx, r.http, r.base+"/revokeChatInviteLink", params)
}

// ─── Bot group membership ──────────────────────────────────────────────────────

// GetMyGroups returns every group and channel the bot is a member of,
// together with the bot's own role in each.
//
// Useful to discover which groups the bot can manage (e.g. create invite links).
//
// Example:
//
//	result, err := bot.Chats.GetMyGroups(ctx)
//	for _, g := range result.Groups {
//	    fmt.Printf("%d (%s) %q → %s\n", g.ChatID, g.Type, g.Title, g.BotRole)
//	    if g.BotRole == kappelas.ParticipantRoleAdmin {
//	        // bot can create invite links, manage members…
//	    }
//	}
func (r *ChatsResource) GetMyGroups(ctx context.Context) (*GetMyGroupsResult, error) {
	return httpPost[*GetMyGroupsResult](ctx, r.http, r.base+"/getMyGroups", struct{}{})
}

package kappelas

import "context"

// CommunitiesResource provides methods to manage communities (members, roles,
// invites, join/group requests). A bot can administer a community only if it is
// admin of that community.
//
// To make someone (a person OR a bot) a community admin, the flow is two steps —
// add as member first, then promote:
//
//	bot.Communities.AddMember(ctx, kappelas.AddCommunityMemberParams{CommunityID: 7, UserID: "uuid", Role: kappelas.ParticipantRoleMember})
//	bot.Communities.PromoteMember(ctx, kappelas.PromoteCommunityMemberParams{CommunityID: 7, UserID: "uuid", Role: kappelas.ParticipantRoleAdmin})
type CommunitiesResource struct {
	http *httpClient
	base string
}

// ─── Types ──────────────────────────────────────────────────────────────────

// Community is a community the bot/user belongs to.
type Community struct {
	ID                    int64           `json:"id"`
	Name                  string          `json:"name"`
	Description           *string         `json:"description"`
	AvatarURL             *string         `json:"avatar_url"`
	CreatedBy             string          `json:"created_by"`
	AnnouncementChannelID *int64          `json:"announcement_channel_id"`
	RequiresApproval      bool            `json:"requires_approval"`
	CreatedAt             string          `json:"created_at"` // ISO 8601
	// Role of the bot/user IN THE COMMUNITY ("member"|"admin"). Set by List() only.
	// Note: being admin of a GROUP attached to a community does NOT make you admin
	// of the community (distinct scopes).
	Role ParticipantRole `json:"role,omitempty"`
}

// CommunityMember is a member of a community (enriched with name/avatar by Get()).
type CommunityMember struct {
	CommunityID int64           `json:"community_id"`
	UserID      string          `json:"user_id"`
	Role        ParticipantRole `json:"role"`
	JoinedAt    string          `json:"joined_at"`
	Name        string          `json:"name,omitempty"`
	AvatarURL   *string         `json:"avatar_url,omitempty"`
}

// CommunityGroup is a group/channel linked to a community.
type CommunityGroup struct {
	ID                int64   `json:"id"`
	Type              string  `json:"type"`
	Title             *string `json:"title"`
	AvatarURL         *string `json:"avatar_url"`
	Joined            bool    `json:"joined"`
	Pending           bool    `json:"pending"`
	ParticipantsCount int     `json:"participants_count"`
}

// CommunityDetail is the full view returned by Get().
type CommunityDetail struct {
	Community Community         `json:"community"`
	Groups    []CommunityGroup  `json:"groups"`
	Members   []CommunityMember `json:"members"`
}

// CommunityInvite is an invite link to a community.
type CommunityInvite struct {
	Code        string  `json:"code"`
	CommunityID int64   `json:"community_id"`
	CreatedBy   string  `json:"created_by"`
	MaxUses     int     `json:"max_uses"`  // 0 = unlimited, 1+ = capped
	UseCount    int     `json:"use_count"`
	ExpiresAt   *string `json:"expires_at"` // ISO 8601 or nil if permanent
	RevokedAt   *string `json:"revoked_at"` // ISO 8601 or nil if active
	CreatedAt   string  `json:"created_at"`
}

// CommunityInvitePreview is the public preview of a community via an invite code.
type CommunityInvitePreview struct {
	Code          string  `json:"code"`
	CommunityID   int64   `json:"community_id"`
	CommunityName string  `json:"community_name"`
	MemberCount   int     `json:"member_count"`
	ExpiresAt     *string `json:"expires_at"`
	AvatarURL     *string `json:"avatar_url"`
	Description   *string `json:"description"`
}

// CommunityJoinRequest is a pending request from a user to join a community.
type CommunityJoinRequest struct {
	ID                 int64   `json:"id"`
	CommunityID        int64   `json:"community_id"`
	UserID             string  `json:"user_id"`
	Status             string  `json:"status"`
	CreatedAt          string  `json:"created_at"`
	RequesterName      string  `json:"requester_name,omitempty"`
	RequesterAvatarURL *string `json:"requester_avatar_url,omitempty"`
}

// CommunityGroupRequest is a pending request from a group to join a community.
type CommunityGroupRequest struct {
	ID             int64  `json:"id"`
	CommunityID    int64  `json:"community_id"`
	ConversationID int64  `json:"conversation_id"`
	GroupName      string `json:"group_name"`
	RequestedBy    string `json:"requested_by"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

// CommunityActionResult is returned by actions without a body.
// Done is true on success; Pending is true when a join is queued for approval.
type CommunityActionResult struct {
	Done    bool `json:"done"`
	Pending bool `json:"pending"`
}

// GetMyCommunitiesResult wraps the communities list.
type GetMyCommunitiesResult struct {
	Communities []Community `json:"communities"`
}

// GetCommunityInviteLinksResult wraps the invite links list.
type GetCommunityInviteLinksResult struct {
	Invites []CommunityInvite `json:"invites"`
}

// AcceptCommunityInviteResult is returned by AcceptInvite.
type AcceptCommunityInviteResult struct {
	CommunityID int64 `json:"community_id"`
}

// ─── Params ─────────────────────────────────────────────────────────────────

type GetCommunityParams struct {
	CommunityID int64 `json:"community_id"`
}

type CreateCommunityParams struct {
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	AvatarURL        string `json:"avatar_url,omitempty"`
	RequiresApproval bool   `json:"requires_approval,omitempty"`
}

// UpdateCommunityParams updates only the non-nil fields.
type UpdateCommunityParams struct {
	CommunityID           int64   `json:"community_id"`
	Name                  *string `json:"name,omitempty"`
	Description           *string `json:"description,omitempty"`
	AvatarURL             *string `json:"avatar_url,omitempty"`
	AnnouncementChannelID *int64  `json:"announcement_channel_id,omitempty"`
	RequiresApproval      *bool   `json:"requires_approval,omitempty"`
}

type AddCommunityMemberParams struct {
	CommunityID int64           `json:"community_id"`
	UserID      string          `json:"user_id"`
	Role        ParticipantRole `json:"role,omitempty"` // default "member"
}

type PromoteCommunityMemberParams struct {
	CommunityID int64           `json:"community_id"`
	UserID      string          `json:"user_id"`
	Role        ParticipantRole `json:"role"` // "admin" promotes, "member" demotes
}

type BanCommunityMemberParams struct {
	CommunityID int64  `json:"community_id"`
	UserID      string `json:"user_id"`
}

type CreateCommunityInviteLinkParams struct {
	CommunityID int64  `json:"community_id"`
	MaxUses     int    `json:"max_uses,omitempty"`   // 0 = unlimited (default)
	ExpiresIn   string `json:"expires_in,omitempty"` // "1h"|"24h"|"7d"|"30d"|"never"
}

type RevokeCommunityInviteLinkParams struct {
	CommunityID int64  `json:"community_id"`
	Code        string `json:"code"`
}

type CommunityInviteCodeParams struct {
	Code string `json:"code"`
}

type CommunityRequestActionParams struct {
	CommunityID int64 `json:"community_id"`
	RequestID   int64 `json:"request_id"`
}

type AddCommunityGroupParams struct {
	CommunityID    int64 `json:"community_id"`
	ConversationID int64 `json:"conversation_id"`
}

type RemoveCommunityGroupParams struct {
	CommunityID    int64 `json:"community_id"`
	ConversationID int64 `json:"conversation_id"`
}

// ─── CRUD / listing ─────────────────────────────────────────────────────────

// List returns the communities the bot is a member of (each with the bot's Role).
func (r *CommunitiesResource) List(ctx context.Context) (GetMyCommunitiesResult, error) {
	return httpPost[GetMyCommunitiesResult](ctx, r.http, r.base+"/getMyCommunities", struct{}{})
}

// ListAdmin returns only the communities where the bot is community admin.
//
// Note: this is the role IN THE COMMUNITY — being admin of a group attached to a
// community does NOT make the bot admin of the community.
func (r *CommunitiesResource) ListAdmin(ctx context.Context) ([]Community, error) {
	res, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Community, 0, len(res.Communities))
	for _, c := range res.Communities {
		if c.Role == ParticipantRoleAdmin {
			out = append(out, c)
		}
	}
	return out, nil
}

// Get returns the full detail of a community (infos + groups + members).
func (r *CommunitiesResource) Get(ctx context.Context, params GetCommunityParams) (*CommunityDetail, error) {
	return httpPost[*CommunityDetail](ctx, r.http, r.base+"/getCommunity", params)
}

// Create creates a community (the bot becomes admin).
func (r *CommunitiesResource) Create(ctx context.Context, params CreateCommunityParams) (*Community, error) {
	return httpPost[*Community](ctx, r.http, r.base+"/createCommunity", params)
}

// Update modifies a community (admin). Only non-nil fields are changed.
func (r *CommunitiesResource) Update(ctx context.Context, params UpdateCommunityParams) (*Community, error) {
	return httpPost[*Community](ctx, r.http, r.base+"/updateCommunity", params)
}

// Delete deletes a community (admin).
func (r *CommunitiesResource) Delete(ctx context.Context, params GetCommunityParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/deleteCommunity", params)
}

// Join joins a community. Public → member immediately; approval-required → the
// result has Pending = true (request queued).
func (r *CommunitiesResource) Join(ctx context.Context, params GetCommunityParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/joinCommunity", params)
}

// ─── Members ────────────────────────────────────────────────────────────────

// AddMember adds a member (person or bot). The bot must be community admin.
func (r *CommunitiesResource) AddMember(ctx context.Context, params AddCommunityMemberParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/addCommunityMember", params)
}

// PromoteMember promotes ("admin") or demotes ("member") a member. The bot must
// be community admin. The member must already exist (add it first).
func (r *CommunitiesResource) PromoteMember(ctx context.Context, params PromoteCommunityMemberParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/promoteCommunityMember", params)
}

// BanMember removes a member. The bot must be community admin (or removes itself).
func (r *CommunitiesResource) BanMember(ctx context.Context, params BanCommunityMemberParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/banCommunityMember", params)
}

// Leave makes the bot leave the community.
func (r *CommunitiesResource) Leave(ctx context.Context, params GetCommunityParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/leaveCommunity", params)
}

// ─── Invite links ───────────────────────────────────────────────────────────

// CreateInviteLink creates an invite link. The bot must be community admin.
func (r *CommunitiesResource) CreateInviteLink(ctx context.Context, params CreateCommunityInviteLinkParams) (*CommunityInvite, error) {
	return httpPost[*CommunityInvite](ctx, r.http, r.base+"/createCommunityInviteLink", params)
}

// GetInviteLinks returns the active invite links. The bot must be community admin.
func (r *CommunitiesResource) GetInviteLinks(ctx context.Context, params GetCommunityParams) (*GetCommunityInviteLinksResult, error) {
	return httpPost[*GetCommunityInviteLinksResult](ctx, r.http, r.base+"/getCommunityInviteLinks", params)
}

// RevokeInviteLink revokes an invite link. The bot must be community admin.
func (r *CommunitiesResource) RevokeInviteLink(ctx context.Context, params RevokeCommunityInviteLinkParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/revokeCommunityInviteLink", params)
}

// PreviewInvite returns the public preview of a community from an invite code.
func (r *CommunitiesResource) PreviewInvite(ctx context.Context, params CommunityInviteCodeParams) (*CommunityInvitePreview, error) {
	return httpPost[*CommunityInvitePreview](ctx, r.http, r.base+"/previewCommunityInvite", params)
}

// AcceptInvite makes the bot join a community via an invite code.
func (r *CommunitiesResource) AcceptInvite(ctx context.Context, params CommunityInviteCodeParams) (*AcceptCommunityInviteResult, error) {
	return httpPost[*AcceptCommunityInviteResult](ctx, r.http, r.base+"/acceptCommunityInvite", params)
}

// ─── Join requests (user → community) ───────────────────────────────────────

// GetJoinRequests returns pending join requests (approval-required mode). Admin.
func (r *CommunitiesResource) GetJoinRequests(ctx context.Context, params GetCommunityParams) ([]CommunityJoinRequest, error) {
	reqs, err := httpPost[[]CommunityJoinRequest](ctx, r.http, r.base+"/getCommunityJoinRequests", params)
	if err != nil {
		return nil, err
	}
	if reqs == nil {
		reqs = []CommunityJoinRequest{}
	}
	return reqs, nil
}

// ApproveJoinRequest approves a join request. Admin.
func (r *CommunitiesResource) ApproveJoinRequest(ctx context.Context, params CommunityRequestActionParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/approveCommunityJoinRequest", params)
}

// RejectJoinRequest rejects a join request. Admin.
func (r *CommunitiesResource) RejectJoinRequest(ctx context.Context, params CommunityRequestActionParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/rejectCommunityJoinRequest", params)
}

// ─── Group requests + linking groups ────────────────────────────────────────

// GetGroupRequests returns pending requests from groups to join the community. Admin.
func (r *CommunitiesResource) GetGroupRequests(ctx context.Context, params GetCommunityParams) ([]CommunityGroupRequest, error) {
	reqs, err := httpPost[[]CommunityGroupRequest](ctx, r.http, r.base+"/getCommunityGroupRequests", params)
	if err != nil {
		return nil, err
	}
	if reqs == nil {
		reqs = []CommunityGroupRequest{}
	}
	return reqs, nil
}

// ApproveGroupRequest approves a group request (links the group). Admin.
func (r *CommunitiesResource) ApproveGroupRequest(ctx context.Context, params CommunityRequestActionParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/approveCommunityGroupRequest", params)
}

// RejectGroupRequest rejects a group request. Admin.
func (r *CommunitiesResource) RejectGroupRequest(ctx context.Context, params CommunityRequestActionParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/rejectCommunityGroupRequest", params)
}

// AddGroup links a group to the community (the bot must be community admin AND group admin).
func (r *CommunitiesResource) AddGroup(ctx context.Context, params AddCommunityGroupParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/addCommunityGroup", params)
}

// RemoveGroup unlinks a group from the community (community admin OR group admin).
func (r *CommunitiesResource) RemoveGroup(ctx context.Context, params RemoveCommunityGroupParams) (*CommunityActionResult, error) {
	return httpPost[*CommunityActionResult](ctx, r.http, r.base+"/removeCommunityGroup", params)
}

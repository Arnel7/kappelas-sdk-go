package kappelas

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Vérifie que chaque méthode communities tape le bon endpoint /v1/{token}/{method}
// avec le bon payload, et que les résultats sont bien désérialisés. Aucun réseau réel.
func TestCommunities(t *testing.T) {
	const base = "/v1/test-token"
	var lastPath string
	var lastBody map[string]any
	var result any = map[string]any{"done": true}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		lastBody = nil
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			_ = json.Unmarshal(b, &lastBody)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
	}))
	defer srv.Close()

	bot := NewBot("test-token", WithBaseURL(srv.URL))
	ctx := context.Background()

	jsonEq := func(a, b any) bool {
		x, _ := json.Marshal(a)
		y, _ := json.Marshal(b)
		return string(x) == string(y)
	}
	expect := func(name, method string, want map[string]any, call func() error) {
		lastPath, lastBody = "", nil
		if err := call(); err != nil {
			t.Fatalf("%s: erreur %v", name, err)
		}
		if lastPath != base+"/"+method {
			t.Errorf("%s: endpoint = %s, attendu %s/%s", name, lastPath, base, method)
		}
		if want != nil && !jsonEq(lastBody, want) {
			gx, _ := json.Marshal(lastBody)
			t.Errorf("%s: payload = %s, attendu %v", name, gx, want)
		}
	}

	cid := map[string]any{"community_id": 7}

	expect("List", "getMyCommunities", map[string]any{}, func() error { _, e := bot.Communities.List(ctx); return e })
	expect("Get", "getCommunity", cid, func() error { _, e := bot.Communities.Get(ctx, GetCommunityParams{CommunityID: 7}); return e })
	expect("Create", "createCommunity", map[string]any{"name": "Devs", "requires_approval": true}, func() error {
		_, e := bot.Communities.Create(ctx, CreateCommunityParams{Name: "Devs", RequiresApproval: true})
		return e
	})
	expect("Delete", "deleteCommunity", cid, func() error { _, e := bot.Communities.Delete(ctx, GetCommunityParams{CommunityID: 7}); return e })
	expect("Join", "joinCommunity", cid, func() error { _, e := bot.Communities.Join(ctx, GetCommunityParams{CommunityID: 7}); return e })
	expect("AddMember", "addCommunityMember", map[string]any{"community_id": 7, "user_id": "u", "role": "member"}, func() error {
		_, e := bot.Communities.AddMember(ctx, AddCommunityMemberParams{CommunityID: 7, UserID: "u", Role: ParticipantRoleMember})
		return e
	})
	expect("PromoteMember", "promoteCommunityMember", map[string]any{"community_id": 7, "user_id": "u", "role": "admin"}, func() error {
		_, e := bot.Communities.PromoteMember(ctx, PromoteCommunityMemberParams{CommunityID: 7, UserID: "u", Role: ParticipantRoleAdmin})
		return e
	})
	expect("BanMember", "banCommunityMember", map[string]any{"community_id": 7, "user_id": "u"}, func() error {
		_, e := bot.Communities.BanMember(ctx, BanCommunityMemberParams{CommunityID: 7, UserID: "u"})
		return e
	})
	expect("Leave", "leaveCommunity", cid, func() error { _, e := bot.Communities.Leave(ctx, GetCommunityParams{CommunityID: 7}); return e })
	expect("CreateInviteLink", "createCommunityInviteLink", map[string]any{"community_id": 7, "max_uses": 1, "expires_in": "24h"}, func() error {
		_, e := bot.Communities.CreateInviteLink(ctx, CreateCommunityInviteLinkParams{CommunityID: 7, MaxUses: 1, ExpiresIn: "24h"})
		return e
	})
	expect("GetInviteLinks", "getCommunityInviteLinks", cid, func() error { _, e := bot.Communities.GetInviteLinks(ctx, GetCommunityParams{CommunityID: 7}); return e })
	expect("RevokeInviteLink", "revokeCommunityInviteLink", map[string]any{"community_id": 7, "code": "c"}, func() error {
		_, e := bot.Communities.RevokeInviteLink(ctx, RevokeCommunityInviteLinkParams{CommunityID: 7, Code: "c"})
		return e
	})
	expect("PreviewInvite", "previewCommunityInvite", map[string]any{"code": "c"}, func() error {
		_, e := bot.Communities.PreviewInvite(ctx, CommunityInviteCodeParams{Code: "c"})
		return e
	})
	expect("AcceptInvite", "acceptCommunityInvite", map[string]any{"code": "c"}, func() error {
		_, e := bot.Communities.AcceptInvite(ctx, CommunityInviteCodeParams{Code: "c"})
		return e
	})

	// Demandes — résultat tableau
	result = []any{}
	expect("GetJoinRequests", "getCommunityJoinRequests", cid, func() error { _, e := bot.Communities.GetJoinRequests(ctx, GetCommunityParams{CommunityID: 7}); return e })
	expect("GetGroupRequests", "getCommunityGroupRequests", cid, func() error { _, e := bot.Communities.GetGroupRequests(ctx, GetCommunityParams{CommunityID: 7}); return e })
	result = map[string]any{"done": true}

	reqBody := map[string]any{"community_id": 7, "request_id": 3}
	expect("ApproveJoinRequest", "approveCommunityJoinRequest", reqBody, func() error { _, e := bot.Communities.ApproveJoinRequest(ctx, CommunityRequestActionParams{CommunityID: 7, RequestID: 3}); return e })
	expect("RejectJoinRequest", "rejectCommunityJoinRequest", reqBody, func() error { _, e := bot.Communities.RejectJoinRequest(ctx, CommunityRequestActionParams{CommunityID: 7, RequestID: 3}); return e })
	expect("ApproveGroupRequest", "approveCommunityGroupRequest", reqBody, func() error { _, e := bot.Communities.ApproveGroupRequest(ctx, CommunityRequestActionParams{CommunityID: 7, RequestID: 3}); return e })
	expect("RejectGroupRequest", "rejectCommunityGroupRequest", reqBody, func() error { _, e := bot.Communities.RejectGroupRequest(ctx, CommunityRequestActionParams{CommunityID: 7, RequestID: 3}); return e })
	expect("AddGroup", "addCommunityGroup", map[string]any{"community_id": 7, "conversation_id": 9}, func() error { _, e := bot.Communities.AddGroup(ctx, AddCommunityGroupParams{CommunityID: 7, ConversationID: 9}); return e })
	expect("RemoveGroup", "removeCommunityGroup", map[string]any{"community_id": 7, "conversation_id": 9}, func() error { _, e := bot.Communities.RemoveGroup(ctx, RemoveCommunityGroupParams{CommunityID: 7, ConversationID: 9}); return e })

	// Update : seuls les champs fournis sont envoyés (omitempty)
	desc := "d"
	expect("Update", "updateCommunity", map[string]any{"community_id": 7, "description": "d"}, func() error {
		_, e := bot.Communities.Update(ctx, UpdateCommunityParams{CommunityID: 7, Description: &desc})
		return e
	})

	// ListAdmin : ne garde que role=admin ; et le résultat est bien désérialisé
	result = map[string]any{"communities": []any{
		map[string]any{"id": 1, "name": "A", "role": "admin"},
		map[string]any{"id": 2, "name": "B", "role": "member"},
		map[string]any{"id": 3, "name": "C", "role": "admin"},
	}}
	admins, err := bot.Communities.ListAdmin(ctx)
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}
	if len(admins) != 2 || admins[0].ID != 1 || admins[1].ID != 3 {
		t.Errorf("ListAdmin: attendu [1,3], obtenu %+v", admins)
	}
}

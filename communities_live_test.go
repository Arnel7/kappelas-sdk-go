package kappelas

import (
	"context"
	"os"
	"testing"
)

// Test LIVE contre le backend déployé. Skippé si KAPPELA_TOKEN n'est pas défini.
//   KAPPELA_TOKEN=xxxx go test -run TestCommunitiesLive -v
func TestCommunitiesLive(t *testing.T) {
	token := os.Getenv("KAPPELA_TOKEN")
	if token == "" {
		t.Skip("KAPPELA_TOKEN non défini — test live ignoré")
	}
	bot := NewBot(token)
	ctx := context.Background()

	c, err := bot.Communities.Create(ctx, CreateCommunityParams{Name: "Go SDK live test (auto)"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Logf("créée id=%d role(création)", c.ID)
	defer func() {
		if _, e := bot.Communities.Delete(ctx, GetCommunityParams{CommunityID: c.ID}); e != nil {
			t.Errorf("Delete cleanup: %v", e)
		} else {
			t.Logf("supprimée id=%d ✓", c.ID)
		}
	}()

	res, err := bot.Communities.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var role ParticipantRole
	for _, x := range res.Communities {
		if x.ID == c.ID {
			role = x.Role
		}
	}
	if role != ParticipantRoleAdmin {
		t.Errorf("List: role attendu admin, obtenu %q", role)
	} else {
		t.Logf("List voit la commu en role=admin ✓")
	}

	inv, err := bot.Communities.CreateInviteLink(ctx, CreateCommunityInviteLinkParams{CommunityID: c.ID, MaxUses: 1, ExpiresIn: "1h"})
	if err != nil {
		t.Fatalf("CreateInviteLink: %v", err)
	}
	t.Logf("invite code=%s ✓", inv.Code)
	if _, err := bot.Communities.RevokeInviteLink(ctx, RevokeCommunityInviteLinkParams{CommunityID: c.ID, Code: inv.Code}); err != nil {
		t.Errorf("RevokeInviteLink: %v", err)
	} else {
		t.Logf("revoke ✓")
	}
}

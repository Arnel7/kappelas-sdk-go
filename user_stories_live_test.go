package kappelas

import (
	"context"
	"io"
	"net/http"
	"os"
	"testing"
)

// Test LIVE de l'API user (stories + parité). Skippé si KAPPELA_API_KEY non défini.
//
//	KAPPELA_API_KEY=sk_xxx go test -run TestUserLive -v
func TestUserLive(t *testing.T) {
	key := os.Getenv("KAPPELA_API_KEY")
	if key == "" {
		t.Skip("KAPPELA_API_KEY non défini — test live ignoré")
	}
	me := NewUser(key)
	ctx := context.Background()

	// Profil
	if p, err := me.Profile.Get(ctx); err != nil {
		t.Fatalf("Profile.Get: %v", err)
	} else {
		t.Logf("getMe: %s", p.Username)
	}

	// Stories — lectures
	if _, err := me.Stories.GetPreferences(ctx); err != nil {
		t.Errorf("Stories.GetPreferences: %v", err)
	}
	if _, err := me.Stories.ListMine(ctx); err != nil {
		t.Errorf("Stories.ListMine: %v", err)
	}
	if _, err := me.Stories.List(ctx); err != nil {
		t.Errorf("Stories.List(feed): %v", err)
	}

	// Story texte : create → get → getViewers → delete
	st, err := me.Stories.Create(ctx, CreateStoryParams{Type: StoryText, Caption: "Go SDK live test", Audience: StoryAudienceAll})
	if err != nil {
		t.Fatalf("Stories.Create(text): %v", err)
	}
	t.Logf("story créée id=%s", st.ID)
	if _, err := me.Stories.Get(ctx, st.ID); err != nil {
		t.Errorf("Stories.Get: %v", err)
	}
	if _, err := me.Stories.GetViewers(ctx, st.ID); err != nil {
		t.Errorf("Stories.GetViewers: %v", err)
	}
	if _, err := me.Stories.Delete(ctx, st.ID); err != nil {
		t.Errorf("Stories.Delete: %v", err)
	} else {
		t.Logf("story supprimée id=%s ✓", st.ID)
	}

	// Story image : fetch bytes → upload auto via Create(Media) → delete
	if resp, err := http.Get("https://picsum.photos/400/600"); err != nil {
		t.Logf("skip image story (fetch image: %v)", err)
	} else {
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		img, err := me.Stories.Create(ctx, CreateStoryParams{
			Type:    StoryImage,
			Media:   &FileInput{Data: data, Filename: "test.jpg", ContentType: "image/jpeg"},
			Caption: "Go image story",
		})
		if err != nil {
			t.Errorf("Stories.Create(image): %v", err)
		} else {
			t.Logf("story image créée id=%s media=%s", img.ID, img.MediaID)
			if _, err := me.Stories.Delete(ctx, img.ID); err != nil {
				t.Errorf("Stories.Delete(image): %v", err)
			} else {
				t.Logf("story image supprimée ✓")
			}
		}
	}

	// Parité : communautés + chats
	if _, err := me.Communities.List(ctx); err != nil {
		t.Errorf("Communities.List: %v", err)
	}
	if res, err := me.Chats.List(ctx, GetChatsParams{Limit: 5}); err != nil {
		t.Errorf("Chats.List: %v", err)
	} else {
		t.Logf("chats: %d", len(res.Chats))
	}
}

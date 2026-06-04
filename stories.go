package kappelas

import (
	"context"
	"fmt"
)

// StoriesResource provides methods to create and manage stories (ephemeral, 24 h).
//
// Stories are a user-only feature (their audience is based on your private
// conversation contacts) — available on User, not Bot. For image/video stories,
// the SDK uploads the file automatically (like Messages.SendPhoto) and uses the
// resulting media id.
type StoriesResource struct {
	http *httpClient
	base string
}

// ─── Types ──────────────────────────────────────────────────────────────────

// Story media types.
const (
	StoryImage = "image"
	StoryVideo = "video"
	StoryText  = "text"
	StoryPoll  = "poll"
)

// Story audiences.
const (
	StoryAudienceAll      = "all"
	StoryAudienceSelected = "selected"
	StoryAudienceExcluded = "excluded"
)

// Story is an ephemeral story (24 h).
type Story struct {
	ID              string   `json:"id"`
	UserID          string   `json:"user_id"`
	MediaID         string   `json:"media_id"`
	MediaType       string   `json:"media_type"`
	Caption         string   `json:"caption"`
	ExpiresAt       string   `json:"expires_at"` // ISO 8601
	ViewCount       int      `json:"view_count"`
	CreatedAt       string   `json:"created_at"` // ISO 8601
	Audience        string   `json:"audience"`
	AudienceUserIDs []string `json:"audience_user_ids,omitempty"`
	// Enriched on read.
	AuthorName   string  `json:"author_name,omitempty"`
	AuthorAvatar *string `json:"author_avatar,omitempty"`
	ViewedByMe   bool    `json:"viewed_by_me,omitempty"`
	MediaURL     *string `json:"media_url,omitempty"`
}

// StoryView is a single view of a story (enriched with viewer name/avatar).
type StoryView struct {
	StoryID      string  `json:"story_id"`
	ViewerID     string  `json:"viewer_id"`
	ViewedAt     string  `json:"viewed_at"` // ISO 8601
	ViewerName   string  `json:"viewer_name,omitempty"`
	ViewerAvatar *string `json:"viewer_avatar,omitempty"`
}

// StoryMediaUpload is returned when uploading story media (image/video).
type StoryMediaUpload struct {
	MediaID      string `json:"media_id"`
	URL          string `json:"url"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	MediumURL    string `json:"medium_url,omitempty"`
}

// StoryPreferences holds the default audience preference for new stories.
type StoryPreferences struct {
	Audience        string   `json:"audience"`
	AudienceUserIDs []string `json:"audience_user_ids"`
}

// StoryActionResult is returned by actions without a body (delete, view, set preferences).
type StoryActionResult struct {
	Done bool `json:"done"`
}

// CreateStoryParams holds the parameters for creating a story.
type CreateStoryParams struct {
	// Type is one of StoryImage, StoryVideo, StoryText, StoryPoll.
	Type string
	// Media is the image/video file — uploaded automatically by the SDK. Used for
	// image/video stories when MediaID is empty. Ignored for text/poll.
	Media *FileInput
	// MediaID is an alternative to Media: an already-uploaded media id.
	MediaID string
	Caption string
	// Audience is one of StoryAudienceAll (default), StoryAudienceSelected, StoryAudienceExcluded.
	Audience string
	// AudienceUserIDs is required when Audience is selected or excluded.
	AudienceUserIDs []string
}

// ─── Methods ──────────────────────────────────────────────────────────────────

// Create creates a story. For image/video, pass Media (uploaded automatically)
// or a pre-uploaded MediaID. For text/poll, just set Caption.
func (r *StoriesResource) Create(ctx context.Context, params CreateStoryParams) (*Story, error) {
	mediaID := params.MediaID
	if (params.Type == StoryImage || params.Type == StoryVideo) && mediaID == "" {
		if params.Media == nil {
			return nil, fmt.Errorf("kappelas: Create: Media or MediaID is required for image/video stories")
		}
		uploaded, err := r.UploadMedia(ctx, *params.Media)
		if err != nil {
			return nil, err
		}
		mediaID = uploaded.MediaID
	}
	body := map[string]any{
		"media_type": params.Type,
	}
	if mediaID != "" {
		body["media_id"] = mediaID
	}
	if params.Caption != "" {
		body["caption"] = params.Caption
	}
	if params.Audience != "" {
		body["audience"] = params.Audience
	}
	if params.AudienceUserIDs != nil {
		body["audience_user_ids"] = params.AudienceUserIDs
	}
	return httpPost[*Story](ctx, r.http, r.base+"/createStory", body)
}

// UploadMedia uploads story media (image/video) and returns its media id.
// Usually you don't call this directly — Create with Media does it for you.
func (r *StoriesResource) UploadMedia(ctx context.Context, file FileInput) (*StoryMediaUpload, error) {
	ff := formFile{
		fieldName:   "file",
		filename:    file.Filename,
		contentType: file.ContentType,
		data:        file.Data,
	}
	return httpPostForm[*StoryMediaUpload](ctx, r.http, r.base+"/uploadStoryMedia", ff, map[string]string{})
}

// List returns the feed of your contacts' active stories.
func (r *StoriesResource) List(ctx context.Context) ([]Story, error) {
	return httpPost[[]Story](ctx, r.http, r.base+"/getStories", map[string]any{})
}

// ListMine returns your own stories.
func (r *StoriesResource) ListMine(ctx context.Context) ([]Story, error) {
	return httpPost[[]Story](ctx, r.http, r.base+"/getMyStories", map[string]any{})
}

// Get returns a single story by id (audience-checked server-side).
func (r *StoriesResource) Get(ctx context.Context, storyID string) (*Story, error) {
	return httpPost[*Story](ctx, r.http, r.base+"/getStory", map[string]any{"story_id": storyID})
}

// Delete deletes one of your stories.
func (r *StoriesResource) Delete(ctx context.Context, storyID string) (*StoryActionResult, error) {
	return httpPost[*StoryActionResult](ctx, r.http, r.base+"/deleteStory", map[string]any{"story_id": storyID})
}

// View marks a story as viewed.
func (r *StoriesResource) View(ctx context.Context, storyID string) (*StoryActionResult, error) {
	return httpPost[*StoryActionResult](ctx, r.http, r.base+"/viewStory", map[string]any{"story_id": storyID})
}

// GetViewers lists who viewed one of your stories (owner only).
func (r *StoriesResource) GetViewers(ctx context.Context, storyID string) ([]StoryView, error) {
	return httpPost[[]StoryView](ctx, r.http, r.base+"/getStoryViewers", map[string]any{"story_id": storyID})
}

// GetPreferences returns your default story audience preference.
func (r *StoriesResource) GetPreferences(ctx context.Context) (*StoryPreferences, error) {
	return httpPost[*StoryPreferences](ctx, r.http, r.base+"/getStoryPreferences", map[string]any{})
}

// SetPreferences sets your default story audience preference.
func (r *StoriesResource) SetPreferences(ctx context.Context, prefs StoryPreferences) (*StoryActionResult, error) {
	return httpPost[*StoryActionResult](ctx, r.http, r.base+"/setStoryPreferences", prefs)
}

package kappelas

import "context"

// BotProfileResource provides access to the bot's own profile.
type BotProfileResource struct {
	http *httpClient
	base string
}

// Get returns the bot's own profile.
func (r *BotProfileResource) Get(ctx context.Context) (*BotProfile, error) {
	return httpPost[*BotProfile](ctx, r.http, r.base+"/getMe", struct{}{})
}

// UserProfileResource provides access to the user's own profile.
type UserProfileResource struct {
	http *httpClient
	base string
}

// Get returns your own profile.
func (r *UserProfileResource) Get(ctx context.Context) (*UserProfile, error) {
	return httpGet[*UserProfile](ctx, r.http, r.base+"/getMe")
}

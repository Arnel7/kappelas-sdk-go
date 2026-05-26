package kappelas

import (
	"context"
	"fmt"
)

// ChatsResource provides methods to access chats.
type ChatsResource struct {
	http *httpClient
	base string
}

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
//	    fmt.Println(chat.ChatID)
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

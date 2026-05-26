package kappelas

import (
	"context"
	"encoding/json"
)

// MessagesResource provides methods to send and manage messages.
type MessagesResource struct {
	http *httpClient
	base string
}

// Send sends a text message, with optional inline buttons or keyboard.
func (r *MessagesResource) Send(ctx context.Context, params SendMessageParams) (*SendResult, error) {
	return httpPost[*SendResult](ctx, r.http, r.base+"/sendMessage", params)
}

// SendPhoto sends a photo (image file).
func (r *MessagesResource) SendPhoto(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendPhoto", "photo", params)
}

// SendVideo sends a video file.
func (r *MessagesResource) SendVideo(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendVideo", "video", params)
}

// SendDocument sends a document or generic file.
func (r *MessagesResource) SendDocument(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendDocument", "document", params)
}

// SendAudio sends an audio file.
func (r *MessagesResource) SendAudio(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendAudio", "audio", params)
}

func (r *MessagesResource) sendMedia(ctx context.Context, endpoint, fieldName string, params SendMediaParams) (*SendMediaResult, error) {
	ff := formFile{
		fieldName:   fieldName,
		filename:    params.File.Filename,
		contentType: params.File.ContentType,
		data:        params.File.Data,
	}

	fields := map[string]string{
		"chat_id": int64Field(params.ChatID),
	}
	if params.Caption != "" {
		fields["caption"] = params.Caption
	}
	if params.ReplyToID != nil {
		fields["reply_to_id"] = int64Field(*params.ReplyToID)
	}
	if params.DeletePrevious {
		fields["delete_previous"] = boolField(true)
	}
	if params.ReplyMarkup != nil {
		b, err := json.Marshal(params.ReplyMarkup)
		if err != nil {
			return nil, err
		}
		fields["reply_markup"] = string(b)
	}

	return httpPostForm[*SendMediaResult](ctx, r.http, r.base+endpoint, ff, fields)
}

// SendCarousel sends a product or card carousel.
func (r *MessagesResource) SendCarousel(ctx context.Context, params SendCarouselParams) (*SendCarouselResult, error) {
	return httpPost[*SendCarouselResult](ctx, r.http, r.base+"/sendCarousel", params)
}

// SendTyping shows or hides the typing indicator in a chat.
// IsTyping defaults to true when not set.
func (r *MessagesResource) SendTyping(ctx context.Context, params SendTypingParams) (*TypingResult, error) {
	isTyping := true
	if params.IsTyping != nil {
		isTyping = *params.IsTyping
	}
	return httpPost[*TypingResult](ctx, r.http, r.base+"/sendTyping", map[string]any{
		"chat_id":   params.ChatID,
		"is_typing": isTyping,
	})
}

// Edit edits the text or inline keyboard of a message sent by this bot or user.
func (r *MessagesResource) Edit(ctx context.Context, params EditMessageParams) (*EditMessageResult, error) {
	return httpPost[*EditMessageResult](ctx, r.http, r.base+"/editMessage", params)
}

// Delete deletes a message sent by this bot or user.
func (r *MessagesResource) Delete(ctx context.Context, params DeleteMessageParams) (*DeleteResult, error) {
	return httpPost[*DeleteResult](ctx, r.http, r.base+"/deleteMessage", params)
}

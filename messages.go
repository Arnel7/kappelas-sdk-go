package kappelas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

// CloseWebview remotely closes the in-app WebView opened by an "open_webview" action button on
// the recipient's device(s). Use it when the outcome is confirmed server-side (e.g. a payment
// webhook) instead of relying on the web page calling Kappelas.close(). The event reaches all of
// the recipient's connected devices (personal real-time channel).
func (r *MessagesResource) CloseWebview(ctx context.Context, params CloseWebviewParams) (*CloseWebviewResult, error) {
	return httpPost[*CloseWebviewResult](ctx, r.http, r.base+"/closeWebview", params)
}

// SendPhoto sends a photo (image file).
func (r *MessagesResource) SendPhoto(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendPhoto", "photo", "sending_photo", params)
}

// SendVideo sends a video file.
func (r *MessagesResource) SendVideo(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendVideo", "video", "sending_video", params)
}

// SendDocument sends a document or generic file.
func (r *MessagesResource) SendDocument(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendDocument", "document", "sending_document", params)
}

// SendAudio sends an audio file.
func (r *MessagesResource) SendAudio(ctx context.Context, params SendMediaParams) (*SendMediaResult, error) {
	return r.sendMedia(ctx, "/sendAudio", "audio", "recording_audio", params)
}

// pingTyping fires a best-effort "typing" indicator toward the media
// recipient so they see activity while a potentially slow upload runs.
// It is fire-and-forget: it runs in its own goroutine on a detached context
// and never blocks or affects the outcome of the media send. The action
// (e.g. "sending_photo") tailors the indicator to the media type.
func (r *MessagesResource) pingTyping(action string, params SendMediaParams) {
	isTyping := true
	tp := SendTypingParams{IsTyping: &isTyping, Action: action}
	// Recipient — chat_id takes priority; otherwise route to the user's private chat.
	if params.ChatID != 0 {
		tp.ChatID = params.ChatID
	} else if params.UserID != "" {
		tp.UserID = params.UserID
	} else {
		return
	}
	go func() {
		defer func() { _ = recover() }()
		_, _ = r.SendTyping(context.Background(), tp)
	}()
}

func (r *MessagesResource) sendMedia(ctx context.Context, endpoint, fieldName, action string, params SendMediaParams) (*SendMediaResult, error) {
	r.pingTyping(action, params)

	ff := formFile{
		fieldName:   fieldName,
		filename:    params.File.Filename,
		contentType: params.File.ContentType,
		data:        params.File.Data,
	}

	fields := map[string]string{}
	// Recipient — chat_id takes priority; otherwise route to the user's private chat.
	if params.ChatID != 0 {
		fields["chat_id"] = int64Field(params.ChatID)
	} else if params.UserID != "" {
		fields["user_id"] = params.UserID
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
	body := map[string]any{"is_typing": isTyping}
	// Recipient — chat_id takes priority; otherwise route to the user's private chat.
	if params.ChatID != 0 {
		body["chat_id"] = params.ChatID
	} else if params.UserID != "" {
		body["user_id"] = params.UserID
	}
	if params.Action != "" {
		body["action"] = params.Action
	}
	return httpPost[*TypingResult](ctx, r.http, r.base+"/sendTyping", body)
}

// Edit edits the text or inline keyboard of a message sent by this bot or user.
func (r *MessagesResource) Edit(ctx context.Context, params EditMessageParams) (*EditMessageResult, error) {
	return httpPost[*EditMessageResult](ctx, r.http, r.base+"/editMessage", params)
}

// Delete deletes a message sent by this bot or user.
func (r *MessagesResource) Delete(ctx context.Context, params DeleteMessageParams) (*DeleteResult, error) {
	return httpPost[*DeleteResult](ctx, r.http, r.base+"/deleteMessage", params)
}

// GetFile returns metadata and a short-lived signed download URL for a media file.
func (r *MessagesResource) GetFile(ctx context.Context, mediaID string) (*FileInfo, error) {
	return httpGet[*FileInfo](ctx, r.http, r.base+"/getFile?media_id="+url.QueryEscape(mediaID))
}

// DownloadFile resolves a media file's signed URL via GetFile and downloads its bytes.
func (r *MessagesResource) DownloadFile(ctx context.Context, mediaID string) ([]byte, error) {
	info, err := r.GetFile(ctx, mediaID)
	if err != nil {
		return nil, err
	}
	if info.URL == "" {
		return nil, fmt.Errorf("no download URL for media %s", mediaID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.http.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("downloading file: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

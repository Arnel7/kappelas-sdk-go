package kappelas

import "context"

// WebhooksResource provides methods to manage webhooks.
type WebhooksResource struct {
	http *httpClient
	base string
}

// Set registers a webhook URL. Use this for production deployments.
func (r *WebhooksResource) Set(ctx context.Context, params SetWebhookParams) (*WebhookSetResult, error) {
	return httpPost[*WebhookSetResult](ctx, r.http, r.base+"/setWebhook", params)
}

// GetInfo returns the current webhook status and URL.
func (r *WebhooksResource) GetInfo(ctx context.Context) (*WebhookInfo, error) {
	return httpGet[*WebhookInfo](ctx, r.http, r.base+"/getWebhookInfo")
}

// Delete removes the webhook. Events will no longer be delivered via HTTP POST.
func (r *WebhooksResource) Delete(ctx context.Context) (*WebhookDeleteResult, error) {
	return httpPost[*WebhookDeleteResult](ctx, r.http, r.base+"/deleteWebhook", struct{}{})
}

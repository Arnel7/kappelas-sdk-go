package kappelas

import (
	"fmt"
	"strings"
)

const docsBase = "https://docs.kappelas.com/errors"

type errorHint struct {
	description string
	solutions   []string
	slug        string
}

var hints = map[ErrorCode]errorHint{
	ErrCodeUnauthorized: {
		description: "Authentication failed. Your token or API key is invalid or expired.",
		solutions: []string{
			"Verify your bot token is correct",
			"Ensure your API key has not been revoked",
		},
		slug: "unauthorized",
	},
	ErrCodeForbidden: {
		description: "You do not have permission to perform this action.",
		solutions: []string{
			"Check that your bot is a participant in this chat",
			"Verify you have the required role (e.g. admin)",
		},
		slug: "forbidden",
	},
	ErrCodeNotFound: {
		description: "The requested resource does not exist.",
		solutions: []string{
			"Check the ID is correct",
			"Make sure your bot has access to this resource",
			"List available chats with: bot.Chats.List(ctx, kappelas.GetChatsParams{})",
		},
		slug: "not_found",
	},
	ErrCodeMissingField: {
		description: "A required field is missing from your request.",
		solutions: []string{
			"Check the params struct — all required fields must be set",
			"See the full parameter list at the docs link below",
		},
		slug: "missing_field",
	},
	ErrCodeInvalidField: {
		description: "One or more fields contain invalid values.",
		solutions: []string{
			"Verify field types match the expected types (e.g. ChatID must be an int64)",
			"Check string length and format constraints",
		},
		slug: "invalid_field",
	},
	ErrCodeConflict: {
		description: "The resource already exists or conflicts with an existing state.",
		solutions:   []string{"Check if the resource already exists before creating it"},
		slug:        "conflict",
	},
	ErrCodeInternalError: {
		description: "An unexpected error occurred on the Kappela servers.",
		solutions: []string{
			"Retry the request — this is usually transient",
			"If the problem persists, contact support with the RequestID",
		},
		slug: "internal_error",
	},
	ErrCodeServiceUnavailable: {
		description: "A Kappela service is temporarily unavailable.",
		solutions: []string{
			"Retry with exponential backoff",
			"Check status.kappelas.com for ongoing incidents",
		},
		slug: "service_unavailable",
	},
	ErrCodeUpstreamError: {
		description: "An upstream Kappela service returned an unexpected response.",
		solutions: []string{
			"Retry the request",
			"Check status.kappelas.com for service issues",
		},
		slug: "upstream_error",
	},
	ErrCodeMethodNotAllowed: {
		description: "The HTTP method used is not allowed for this endpoint.",
		solutions: []string{
			"Check you are using the correct HTTP method (GET vs POST)",
			"See the API documentation for this endpoint",
		},
		slug: "method_not_allowed",
	},
	ErrCodeInvalidPath: {
		description: "The requested API path does not exist.",
		solutions: []string{
			"Check for typos in the endpoint path",
			"Verify the SDK version matches the API version",
		},
		slug: "invalid_path",
	},
}

// KappelaError is returned when the Kappela API responds with an error.
type KappelaError struct {
	// Message is the human-readable error description from the API.
	Message string
	// Code is the machine-readable error code.
	Code ErrorCode
	// Status is the HTTP status code.
	Status int
	// RequestID can be quoted when contacting support.
	RequestID string
}

func (e *KappelaError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "KappelaError: %s\n  Code:   %s\n  Status: %d", e.Message, e.Code, e.Status)
	if h, ok := hints[e.Code]; ok {
		fmt.Fprintf(&b, "\n\n  %s\n\n  Possible solutions:", h.description)
		for _, s := range h.solutions {
			fmt.Fprintf(&b, "\n  - %s", s)
		}
		fmt.Fprintf(&b, "\n\n  Docs: %s/%s", docsBase, h.slug)
	}
	if e.RequestID != "" {
		fmt.Fprintf(&b, "\n  Request ID: %s  (mention this when contacting support)", e.RequestID)
	}
	return b.String()
}

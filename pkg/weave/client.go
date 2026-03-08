package weave

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/hironow/weaveback/pkg/weave/gen"
)

const (
	defaultRetryMax     = 3
	defaultRetryWaitMin = 1 * time.Second
	defaultRetryWaitMax = 10 * time.Second
	defaultTimeout      = 30 * time.Second
)

// ClientError represents an API error with HTTP status information.
type ClientError struct {
	StatusCode int
	Body       []byte
	Retryable  bool
}

func (e *ClientError) Error() string {
	return fmt.Sprintf("weave API error: status=%d retryable=%t body=%s", e.StatusCode, e.Retryable, string(e.Body))
}

// AsClientError checks if err is or wraps a *ClientError.
func AsClientError(err error, target **ClientError) bool {
	return errors.As(err, target)
}

// Client wraps the oapi-codegen generated client with authentication and retry logic.
type Client struct {
	inner *gen.ClientWithResponses
}

// NewClient creates a new Weave API client.
// apiKey is used for HTTP Basic authentication (user "api", password apiKey).
func NewClient(serverURL string, apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = defaultRetryMax
	retryClient.RetryWaitMin = defaultRetryWaitMin
	retryClient.RetryWaitMax = defaultRetryWaitMax
	retryClient.Logger = nil // suppress default logging
	retryClient.CheckRetry = retryPolicy
	retryClient.HTTPClient.Timeout = defaultTimeout

	authEditor := func(ctx context.Context, req *http.Request) error {
		req.SetBasicAuth("api", apiKey)
		return nil
	}

	inner, err := gen.NewClientWithResponses(
		serverURL,
		gen.WithHTTPClient(retryClient.StandardClient()),
		gen.WithRequestEditorFn(authEditor),
	)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}

	return &Client{inner: inner}, nil
}

// retryPolicy determines whether a request should be retried.
func retryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	if err != nil {
		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return true, nil
	}
	if resp.StatusCode >= 500 {
		return true, nil
	}
	return false, nil
}

// isRetryableStatus returns true for status codes that should trigger a retry.
func isRetryableStatus(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

// CreateFeedback creates a single feedback entry.
func (c *Client) CreateFeedback(ctx context.Context, req gen.FeedbackCreateReq) (*gen.FeedbackCreateRes, error) {
	resp, err := c.inner.FeedbackCreateFeedbackCreatePostWithResponse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create feedback: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, &ClientError{
			StatusCode: resp.StatusCode(),
			Body:       resp.Body,
			Retryable:  isRetryableStatus(resp.StatusCode()),
		}
	}
	return resp.JSON200, nil
}

// CreateFeedbackBatch creates multiple feedback entries in a single request.
func (c *Client) CreateFeedbackBatch(ctx context.Context, req gen.FeedbackCreateBatchReq) (*gen.FeedbackCreateBatchRes, error) {
	resp, err := c.inner.FeedbackCreateBatchFeedbackBatchCreatePostWithResponse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create feedback batch: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, &ClientError{
			StatusCode: resp.StatusCode(),
			Body:       resp.Body,
			Retryable:  isRetryableStatus(resp.StatusCode()),
		}
	}
	return resp.JSON200, nil
}

// QueryFeedback queries feedback entries.
func (c *Client) QueryFeedback(ctx context.Context, req gen.FeedbackQueryReq) (*gen.FeedbackQueryRes, error) {
	resp, err := c.inner.FeedbackQueryFeedbackQueryPostWithResponse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("query feedback: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, &ClientError{
			StatusCode: resp.StatusCode(),
			Body:       resp.Body,
			Retryable:  isRetryableStatus(resp.StatusCode()),
		}
	}
	return resp.JSON200, nil
}

// PurgeFeedback purges feedback entries matching the query.
func (c *Client) PurgeFeedback(ctx context.Context, req gen.FeedbackPurgeReq) (*gen.FeedbackPurgeRes, error) {
	resp, err := c.inner.FeedbackPurgeFeedbackPurgePostWithResponse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("purge feedback: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, &ClientError{
			StatusCode: resp.StatusCode(),
			Body:       resp.Body,
			Retryable:  isRetryableStatus(resp.StatusCode()),
		}
	}
	return resp.JSON200, nil
}

// ReplaceFeedback replaces an existing feedback entry.
func (c *Client) ReplaceFeedback(ctx context.Context, req gen.FeedbackReplaceReq) (*gen.FeedbackReplaceRes, error) {
	resp, err := c.inner.FeedbackReplaceFeedbackReplacePostWithResponse(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("replace feedback: %w", err)
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, &ClientError{
			StatusCode: resp.StatusCode(),
			Body:       resp.Body,
			Retryable:  isRetryableStatus(resp.StatusCode()),
		}
	}
	return resp.JSON200, nil
}

package weave

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hironow/rest/pkg/weave/gen"
)

var emojiPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// AddReaction creates an emoji reaction feedback for a given call.
// emoji must match ^[a-zA-Z0-9_-]{1,64}$ (e.g. "thumbsup", "heart").
func (c *Client) AddReaction(ctx context.Context, projectID, callID, emoji string) error {
	if !emojiPattern.MatchString(emoji) {
		return fmt.Errorf("invalid emoji: must match %s", emojiPattern.String())
	}

	_, err := c.CreateFeedback(ctx, gen.FeedbackCreateReq{
		ProjectID:    projectID,
		WeaveRef:     callID,
		FeedbackType: "emoji",
		Payload:      map[string]any{"emoji": emoji},
	})
	return err
}

// AddNote creates a comment feedback for a given call.
// note must not be empty.
func (c *Client) AddNote(ctx context.Context, projectID, callID, note string) error {
	if note == "" {
		return fmt.Errorf("invalid note: must not be empty")
	}

	_, err := c.CreateFeedback(ctx, gen.FeedbackCreateReq{
		ProjectID:    projectID,
		WeaveRef:     callID,
		FeedbackType: "comment",
		Payload:      map[string]any{"note": note},
	})
	return err
}

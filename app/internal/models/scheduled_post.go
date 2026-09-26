package models

import "time"

// ScheduledPostStatus is the lifecycle state of a scheduled post.
type ScheduledPostStatus string

const (
	// ScheduledPostPending means the post hasn't been sent yet.
	ScheduledPostPending ScheduledPostStatus = "pending"
	// ScheduledPostPosted means it was sent successfully.
	ScheduledPostPosted ScheduledPostStatus = "posted"
	// ScheduledPostFailed means sending it failed; see the Error field.
	ScheduledPostFailed ScheduledPostStatus = "failed"
	// ScheduledPostCanceled means the user canceled it before it was sent.
	ScheduledPostCanceled ScheduledPostStatus = "canceled"
)

// ScheduledPost is the stored shape from scheduled_posts.
type ScheduledPost struct {
	ID                 string              `json:"id"`
	UserID             string              `json:"user_id,omitempty"`
	StoragePath        string              `json:"storage_path"`
	Filename           string              `json:"filename"`
	CaptionFacebook    string              `json:"caption_facebook"`
	CaptionInstagram   *string             `json:"caption_instagram,omitempty"`
	PostFacebook       bool                `json:"post_facebook"`
	PostInstagram      bool                `json:"post_instagram"`
	CategoryIDs        []string            `json:"category_ids"`
	ScheduledAt        time.Time           `json:"scheduled_at"`
	Status             ScheduledPostStatus `json:"status"`
	ResultSocialPostID *string             `json:"result_social_post_id,omitempty"`
	Error              *string             `json:"error,omitempty"`
	CreatedAt          time.Time           `json:"created_at,omitempty"`
}

// ScheduledPostInput is the write shape for Create.
type ScheduledPostInput struct {
	UserID           string    `json:"user_id,omitempty"`
	StoragePath      string    `json:"storage_path"`
	Filename         string    `json:"filename"`
	CaptionFacebook  string    `json:"caption_facebook"`
	CaptionInstagram *string   `json:"caption_instagram,omitempty"`
	PostFacebook     bool      `json:"post_facebook"`
	PostInstagram    bool      `json:"post_instagram"`
	CategoryIDs      []string  `json:"category_ids"`
	ScheduledAt      time.Time `json:"scheduled_at"`
}

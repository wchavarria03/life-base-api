package models

import "time"

// SocialPostStatus is the outcome of a post attempt against one network.
type SocialPostStatus string

const (
	SocialPostSuccess SocialPostStatus = "success"
	SocialPostFailed  SocialPostStatus = "failed"
	// SocialPostSkipped marks a network the caller didn't select to post to.
	SocialPostSkipped SocialPostStatus = "skipped"
)

// SocialPostLifecycle distinguishes how a social_posts row came to exist.
type SocialPostLifecycle string

const (
	// SocialPostLifecycleDraft is created here, not yet posted or scheduled.
	SocialPostLifecycleDraft SocialPostLifecycle = "draft"
	// SocialPostLifecycleLogged happened on Facebook/Instagram before (or
	// outside) this app — recorded for tracking, no Graph API call made.
	SocialPostLifecycleLogged SocialPostLifecycle = "logged"
	// SocialPostLifecyclePosted went through our Post flow, a real Graph
	// API call.
	SocialPostLifecyclePosted SocialPostLifecycle = "posted"
)

// SocialPostSource distinguishes how a 'posted' row was actually sent —
// meaningless (null) for draft/logged rows.
type SocialPostSource string

const (
	// SocialPostSourceDirect is the in-app "Post" button, immediate.
	SocialPostSourceDirect SocialPostSource = "direct"
	// SocialPostSourceScheduledCron is the hourly GitHub Actions job.
	SocialPostSourceScheduledCron SocialPostSource = "scheduled-cron"
	// SocialPostSourceScheduledManual is the "Check scheduled now" button.
	SocialPostSourceScheduledManual SocialPostSource = "scheduled-manual"
)

// SocialPost is the stored shape from social_posts: one record per upload,
// with independent Facebook/Instagram outcomes since Instagram depends on
// the Facebook leg succeeding first.
type SocialPost struct {
	ID                 string              `json:"id"`
	UserID             string              `json:"user_id,omitempty"`
	Filename           string              `json:"filename"`
	Caption            string              `json:"caption"`
	CaptionInstagram   *string             `json:"caption_instagram,omitempty"`
	FacebookStatus     SocialPostStatus    `json:"facebook_status"`
	FacebookPostID     *string             `json:"facebook_post_id,omitempty"`
	FacebookPhotoURL   *string             `json:"facebook_photo_url,omitempty"`
	FacebookPermalink  *string             `json:"facebook_permalink,omitempty"`
	FacebookError      *string             `json:"facebook_error,omitempty"`
	InstagramStatus    SocialPostStatus    `json:"instagram_status"`
	InstagramMediaID   *string             `json:"instagram_media_id,omitempty"`
	InstagramPermalink *string             `json:"instagram_permalink,omitempty"`
	InstagramError     *string             `json:"instagram_error,omitempty"`
	CreatedAt          time.Time           `json:"created_at,omitempty"`
	CategoryIDs        []string            `json:"category_ids,omitempty"`
	PostStatus         SocialPostLifecycle `json:"post_status"`
	PostedAt           *time.Time          `json:"posted_at,omitempty"`
	Edited             bool                `json:"edited,omitempty"`
	ImageStoragePath   *string             `json:"image_storage_path,omitempty"`
	PostFacebook       bool                `json:"post_facebook"`
	PostInstagram      bool                `json:"post_instagram"`
	Source             *SocialPostSource   `json:"source,omitempty"`
}

// SocialPostInput is the write shape for Create.
type SocialPostInput struct {
	UserID             string              `json:"user_id,omitempty"`
	Filename           string              `json:"filename"`
	Caption            string              `json:"caption"`
	CaptionInstagram   *string             `json:"caption_instagram,omitempty"`
	FacebookStatus     SocialPostStatus    `json:"facebook_status"`
	FacebookPostID     *string             `json:"facebook_post_id,omitempty"`
	FacebookPhotoURL   *string             `json:"facebook_photo_url,omitempty"`
	FacebookPermalink  *string             `json:"facebook_permalink,omitempty"`
	FacebookError      *string             `json:"facebook_error,omitempty"`
	InstagramStatus    SocialPostStatus    `json:"instagram_status"`
	InstagramMediaID   *string             `json:"instagram_media_id,omitempty"`
	InstagramPermalink *string             `json:"instagram_permalink,omitempty"`
	InstagramError     *string             `json:"instagram_error,omitempty"`
	PostStatus         SocialPostLifecycle `json:"post_status,omitempty"`
	PostedAt           *time.Time          `json:"posted_at,omitempty"`
	ImageStoragePath   *string             `json:"image_storage_path,omitempty"`
	PostFacebook       bool                `json:"post_facebook"`
	PostInstagram      bool                `json:"post_instagram"`
	Source             *SocialPostSource   `json:"source,omitempty"`
}

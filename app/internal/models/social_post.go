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

// SocialPost is the stored shape from social_posts: one record per upload,
// with independent Facebook/Instagram outcomes since Instagram depends on
// the Facebook leg succeeding first.
type SocialPost struct {
	ID                 string           `json:"id"`
	UserID             string           `json:"user_id,omitempty"`
	Filename           string           `json:"filename"`
	Caption            string           `json:"caption"`
	FacebookStatus     SocialPostStatus `json:"facebook_status"`
	FacebookPostID     *string          `json:"facebook_post_id,omitempty"`
	FacebookPhotoURL   *string          `json:"facebook_photo_url,omitempty"`
	FacebookPermalink  *string          `json:"facebook_permalink,omitempty"`
	FacebookError      *string          `json:"facebook_error,omitempty"`
	InstagramStatus    SocialPostStatus `json:"instagram_status"`
	InstagramMediaID   *string          `json:"instagram_media_id,omitempty"`
	InstagramPermalink *string          `json:"instagram_permalink,omitempty"`
	InstagramError     *string          `json:"instagram_error,omitempty"`
	CreatedAt          time.Time        `json:"created_at,omitempty"`
}

// SocialPostInput is the write shape for Create.
type SocialPostInput struct {
	UserID             string           `json:"user_id,omitempty"`
	Filename           string           `json:"filename"`
	Caption            string           `json:"caption"`
	FacebookStatus     SocialPostStatus `json:"facebook_status"`
	FacebookPostID     *string          `json:"facebook_post_id,omitempty"`
	FacebookPhotoURL   *string          `json:"facebook_photo_url,omitempty"`
	FacebookPermalink  *string          `json:"facebook_permalink,omitempty"`
	FacebookError      *string          `json:"facebook_error,omitempty"`
	InstagramStatus    SocialPostStatus `json:"instagram_status"`
	InstagramMediaID   *string          `json:"instagram_media_id,omitempty"`
	InstagramPermalink *string          `json:"instagram_permalink,omitempty"`
	InstagramError     *string          `json:"instagram_error,omitempty"`
}

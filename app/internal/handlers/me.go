package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"life-base-api/app/internal/auth"
)

// NewMeHandler constructs a MeHandler.
func NewMeHandler(admin AdminManager, childProfiles ChildProfileManager) *MeHandler {
	return &MeHandler{admin: admin, childProfiles: childProfiles}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	ctx := c.Request.Context()
	userID := auth.UserIDFromContext(ctx)
	role := auth.RoleFromContext(ctx)

	pageKeys, err := h.admin.AllowedPageKeys(ctx, role)
	if err != nil {
		internalError(c, err)
		return
	}

	// A Google-linked kid isn't a household member — AllowedPageKeys above
	// reflects the admin/member role matrix, irrelevant to them. The
	// frontend uses child_profile_id (when set) to render the constrained
	// KidApp shell instead of the normal Sidebar/Navbar, bypassing
	// page_keys entirely for that session.
	var childProfileID *string
	if email := auth.EmailFromContext(ctx); email != "" {
		if profile, err := h.childProfiles.FindMyProfile(ctx, email); err == nil && profile != nil {
			childProfileID = &profile.ID
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":          userID,
		"role":             role,
		"page_keys":        pageKeys,
		"child_profile_id": childProfileID,
	})
}

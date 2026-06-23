package domain

import (
	"errors"
	"time"
)

// Category groups operations for analytics and budgets (BUSINESS_LOGIC.md ф.2, ф.4).
// System categories are seeded per-user at registration; user categories are
// created on demand (manual or auto-categorization).
type Category struct {
	ID        int64
	UserID    int64
	Name      string
	ParentID  *int64
	IsSystem  bool
	CreatedAt time.Time
}

// Validate enforces the category invariants.
func (c Category) Validate() error {
	if c.UserID == 0 {
		return errors.New("category: user_id is required")
	}
	if c.Name == "" {
		return errors.New("category: name is required")
	}
	if c.ParentID != nil && *c.ParentID == 0 {
		return errors.New("category: parent_id must be > 0 when set")
	}
	return nil
}

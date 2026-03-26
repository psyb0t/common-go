package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base is embedded by all models. It provides a UUID
// primary key generated in Go code before insert.
type Base struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
}

// BeforeCreate generates a UUID if one hasn't been
// set. This runs in Go so the ID is available
// immediately after Create without a DB round-trip.
func (b *Base) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}

	return nil
}

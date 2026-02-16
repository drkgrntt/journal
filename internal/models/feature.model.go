package models

import (
	"time"
)

func init() {
	registerModel(&Feature{})
	registerModel(&UserFeature{})
}

type Feature struct {
	*BaseType
	EnabledAt     *time.Time     `gorm:"type:timestamptz" json:"enabledAt,omitempty"`
	Description   string         `gorm:"type:text" json:"description,omitempty"`
	UserFeatures  []*UserFeature `gorm:"foreignKey:FeatureID" json:"userFeatures,omitempty"`
	StripePriceID string         `gorm:"type:text" json:"stripePriceId,omitempty"`
	// Users        []*User        `gorm:"many2many:user_features;" json:"users"`
}

func (f *Feature) IsEnabled() bool {
	return f.EnabledAt != nil
}

type UserFeature struct {
	*Base
	FeatureID int        `gorm:"type:int;not null" json:"featureId"`
	Feature   *Feature   `json:"feature"`
	EnabledAt *time.Time `gorm:"type:timestamptz" json:"enabledAt,omitempty"`
}

func (u *UserFeature) IsEnabled() bool {
	return u.EnabledAt != nil
}

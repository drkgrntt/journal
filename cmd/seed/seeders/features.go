package seeders

import (
	"journal/internal/models"
	"time"
)

func SeedFeatures() error {
	base := &models.Base{
		CreatorID:     &admin.ID,
		LastUpdaterID: &admin.ID,
	}
	now := time.Now().UTC().Add(-time.Hour * 24)
	features := []*models.Feature{
		{
			BaseType:    &models.BaseType{Name: "Routines", Code: "routines", Base: base},
			EnabledAt:   &now,
			Description: "Set regular rhythms in your life. With routines, you can schedule actions to create themselves at regular intervals. Don't worry, if you don't get to it, we won't overwhelm you by piling them up on you. Once you finish one, the next one will come up at the next scheduled time.",
		},
		{
			BaseType:    &models.BaseType{Name: "Custom Topics", Code: "custom-topics", Base: base},
			EnabledAt:   &now,
			Description: "Not seeing what you want to write about in the list we provided? No problem! You decide what buckets you can put your thoughts into.",
		},
	}

	err := db.Create(&features).Error
	return err
}

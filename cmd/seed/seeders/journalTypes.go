package seeders

import (
	"journal/internal/models"
)

func SeedJournalTypes() error {
	base := &models.Base{
		CreatorID:     &admin.ID,
		LastUpdaterID: &admin.ID,
	}
	types := []*models.JournalType{
		{BaseType: &models.BaseType{Name: "General", Code: "general", Base: base}},
		{BaseType: &models.BaseType{Name: "Short Term Goals", Code: "short-term-goals", Base: base}},
		{BaseType: &models.BaseType{Name: "Long Term Goals", Code: "long-term-goals", Base: base}},
		{BaseType: &models.BaseType{Name: "Family", Code: "family", Base: base}},
		{BaseType: &models.BaseType{Name: "Friends", Code: "friends", Base: base}},
		{BaseType: &models.BaseType{Name: "Professional", Code: "professional", Base: base}},
		{BaseType: &models.BaseType{Name: "Hobbies", Code: "hobbies", Base: base}},
		{BaseType: &models.BaseType{Name: "Creative", Code: "creative", Base: base}},
		{BaseType: &models.BaseType{Name: "Health", Code: "health", Base: base}},
		{BaseType: &models.BaseType{Name: "Spiritual", Code: "spiritual", Base: base}},
		{BaseType: &models.BaseType{Name: "Custom", Code: "custom", Base: base}},
	}

	err := db.Create(&types).Error
	return err
}

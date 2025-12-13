package seeders

import (
	"go-starter/internal/models"
)

func seedJournalTypes() error {
	types := []*models.JournalType{
		{BaseType: &models.BaseType{Name: "General", Code: "general"}},
		{BaseType: &models.BaseType{Name: "Short Term Goals", Code: "short-term-goals"}},
		{BaseType: &models.BaseType{Name: "Long Term Goals", Code: "long-term-goals"}},
		{BaseType: &models.BaseType{Name: "Family", Code: "family"}},
		{BaseType: &models.BaseType{Name: "Friends", Code: "friends"}},
		{BaseType: &models.BaseType{Name: "Professional", Code: "professional"}},
		{BaseType: &models.BaseType{Name: "Hobbies", Code: "hobbies"}},
		{BaseType: &models.BaseType{Name: "Creative", Code: "creative"}},
	}

	err := db.Create(&types).Error
	return err
}

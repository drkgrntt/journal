package seeders

import (
	"journal/internal/models"
)

func SeedRatings() error {
	base := &models.Base{
		CreatorID:     &admin.ID,
		LastUpdaterID: &admin.ID,
	}
	ratings := []*models.Rating{
		{BaseType: &models.BaseType{Name: "Awful", Code: "awful", Base: base}, Value: 1},
		{BaseType: &models.BaseType{Name: "Bad", Code: "bad", Base: base}, Value: 2},
		{BaseType: &models.BaseType{Name: "Fine", Code: "fine", Base: base}, Value: 3},
		{BaseType: &models.BaseType{Name: "Good", Code: "good", Base: base}, Value: 4},
		{BaseType: &models.BaseType{Name: "Great", Code: "great", Base: base}, Value: 5},
	}

	err := db.Create(&ratings).Error
	return err
}

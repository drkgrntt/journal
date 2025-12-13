package seeders

import (
	"go-starter/internal/models"
)

func seedRatings() error {
	ratings := []*models.Rating{
		{BaseType: &models.BaseType{Name: "Awful", Code: "awful"}, Value: 1},
		{BaseType: &models.BaseType{Name: "Bad", Code: "bad"}, Value: 2},
		{BaseType: &models.BaseType{Name: "Fine", Code: "fine"}, Value: 3},
		{BaseType: &models.BaseType{Name: "Good", Code: "good"}, Value: 4},
		{BaseType: &models.BaseType{Name: "Great", Code: "great"}, Value: 5},
	}

	err := db.Create(&ratings).Error
	return err
}

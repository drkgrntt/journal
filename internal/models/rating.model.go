package models

func init() {
	registerModel(&Rating{})
}

type Rating struct {
	*BaseType

	Value    int        `gorm:"type:int;not null" json:"value,omitempty"`
	Journals []*Journal `gorm:"foreignKey:RatingID" json:"journals,omitempty"`
}

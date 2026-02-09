package models

func init() {
	registerModel(&CustomJournalType{})
}

type CustomJournalType struct {
	*Base
	Name     string     `gorm:"type:varchar(32);not null;unique" json:"name,omitempty"`
	Journals []*Journal `gorm:"foreignKey:CustomJournalTypeID" json:"journals,omitempty"`
}

package models

func init() {
	registerModel(&CustomJournalType{})
}

type CustomJournalType struct {
	*Base
	Name     string     `gorm:"type:varchar(32);not null;uniqueIndex:idx_custom_journal_types_name,where:deleted_at IS NULL" json:"name,omitempty"`
	Journals []*Journal `gorm:"foreignKey:CustomJournalTypeID" json:"journals,omitempty"`
}

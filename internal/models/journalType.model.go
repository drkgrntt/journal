package models

func init() {
	registerModel(&JournalType{})
}

type JournalType struct {
	*BaseType

	Journals []Journal `gorm:"foreignKey:JournalTypeID" json:"journals,omitempty"`
}

func (j *JournalType) IsDefault() bool {
	return j.Code == "general"
}

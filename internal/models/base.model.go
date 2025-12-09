package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Base default model for basically everything. Excludes ID to allow each model to choose between int and uuid
type Base struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:uuid_generate_v4();primary_key" json:"id,omitempty"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"createdAt,omitempty"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updatedAt,omitempty"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	CreatorID     uuid.UUID      `gorm:"type:uuid;not null" json:"creatorId"`
	Creator       *User          `gorm:"not null" json:"creator"`
	LastUpdaterID uuid.UUID      `gorm:"type:uuid;not null" json:"lastUpdaterId"`
	LastUpdater   *User          `gorm:"not null" json:"lastUpdater"`
	Metadata      datatypes.JSON `gorm:"type:jsonb;default:'{}';" json:"metadata,omitempty"`
}

func (b *Base) SubID() string {
	id := b.ID.String()
	pieces := strings.Split(id, "-")
	return pieces[len(pieces)-1]
}

func CastMetadata[T any](metadata datatypes.JSON) (error, T) {
	var value T

	jsonValue, err := metadata.MarshalJSON()
	if err != nil {
		return err, value
	}

	err = json.Unmarshal(jsonValue, &value)

	return err, value
}

// Base model for types. This includes an auto increment ID, a code and a name, as well as all the base type fields.
type BaseType struct {
	*Base
	ID   int    `gorm:"type:int;autoIncrement:true;primary_key" json:"id,omitempty"`
	Code string `gorm:"type:varchar(32);not null;unique" json:"code,omitempty"`
	Name string `gorm:"type:varchar(32);not null;unique" json:"name,omitempty"`
}

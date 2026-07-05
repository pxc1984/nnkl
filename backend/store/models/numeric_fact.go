package models

import "time"

// NumericFact хранит извлечённый из документа числовой факт.
type NumericFact struct {
	ID         string    `gorm:"type:uuid;primaryKey" json:"id"`
	DocumentID string    `gorm:"type:uuid;index;not null" json:"documentId"`
	ChunkID    string    `gorm:"index" json:"chunkId,omitempty"`
	EntityName string    `gorm:"index" json:"entityName,omitempty"`
	Property   string    `gorm:"index;not null" json:"property"`
	Value      float64   `gorm:"not null" json:"value"`
	Value2     float64   `json:"value2,omitempty"`
	Unit       string    `gorm:"index" json:"unit,omitempty"`
	Operator   string    `json:"operator,omitempty"`
	RawText    string    `json:"rawText,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

func (NumericFact) TableName() string {
	return "numeric_facts"
}

// NumericFactFilter используется для фильтрации фактов в БД.
type NumericFactFilter struct {
	DocumentID string
	Property   string
	Min        float64
	Max        float64
	Unit       string
}

package models

import "github.com/shopspring/decimal"

// ToothMeasurement records the mesiodistal width of a single tooth for a
// patient. ToothNumber uses the FDI/Universal numbering system (a small
// integer), hence tinyint in the schema.
type ToothMeasurement struct {
	Timestamped
	PatientID          uint             `gorm:"not null" json:"patient_id"`
	ToothNumber        int8             `gorm:"type:tinyint;not null" json:"tooth_number"`
	MesiodistalWidthMm *decimal.Decimal `gorm:"type:decimal(4,2)" json:"mesiodistal_width_mm,omitempty"`

	Patient *Patient `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

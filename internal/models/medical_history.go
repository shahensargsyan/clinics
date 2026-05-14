package models

type MedicalHistory struct {
	Base
	PatientID     uint    `gorm:"not null" json:"patient_id"`
	ConditionName string  `gorm:"size:255;not null" json:"condition_name"`
	Notes         *string `gorm:"type:text" json:"notes,omitempty"`

	Patient *Patient `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

// TableName overrides GORM's default pluralization. The Laravel migration
// intentionally kept the singular form `medical_history`, not the default
// `medical_histories`.
func (MedicalHistory) TableName() string { return "medical_history" }

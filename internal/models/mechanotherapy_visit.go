package models

import "gorm.io/datatypes"

type MechanotherapyVisit struct {
	Timestamped
	PatientID       uint            `gorm:"not null" json:"patient_id"`
	VisitDate       *datatypes.Date `gorm:"type:date" json:"visit_date,omitempty"`
	ProcedureNotes  *string         `gorm:"type:text" json:"procedure_notes,omitempty"`
	Recommendations *string         `gorm:"type:text" json:"recommendations,omitempty"`
	DoctorSignature *string         `gorm:"size:100" json:"doctor_signature,omitempty"`

	Patient *Patient `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

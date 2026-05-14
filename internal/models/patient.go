package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Gender string

const (
	GenderMale   Gender = "Male"
	GenderFemale Gender = "Female"
	GenderOther  Gender = "Other"
)

type Patient struct {
	Base
	FirstName   string          `gorm:"size:100;not null" json:"first_name"`
	LastName    string          `gorm:"size:100;not null" json:"last_name"`
	DateOfBirth *datatypes.Date `gorm:"type:date" json:"date_of_birth,omitempty"`
	Gender      *Gender         `gorm:"type:enum('Male','Female','Other')" json:"gender,omitempty"`
	Phone       *string         `gorm:"size:20" json:"phone,omitempty"`
	Email       *string         `gorm:"size:100;uniqueIndex:patients_email_unique" json:"email,omitempty"`
	Address     *string         `gorm:"size:255" json:"address,omitempty"`

	// FullName mirrors the Laravel accessor $patient->full_name; populated
	// on read via AfterFind so frontend payload shape stays identical.
	FullName string `gorm:"-" json:"full_name"`

	Appointments                 []Appointment                `gorm:"foreignKey:PatientID" json:"appointments,omitempty"`
	MedicalHistory               []MedicalHistory             `gorm:"foreignKey:PatientID" json:"medical_history,omitempty"`
	CephalometricAnalyses        []CephalometricAnalysis      `gorm:"foreignKey:PatientID" json:"cephalometric_analyses,omitempty"`
	DiagnosticAssets             []DiagnosticAsset            `gorm:"foreignKey:PatientID" json:"diagnostic_assets,omitempty"`
	MechanotherapyVisits         []MechanotherapyVisit        `gorm:"foreignKey:PatientID" json:"mechanotherapy_visits,omitempty"`
	ToothMeasurements            []ToothMeasurement           `gorm:"foreignKey:PatientID" json:"tooth_measurements,omitempty"`
	OrthodonticsMedicalHistories []OrthodonticsMedicalHistory `gorm:"foreignKey:PatientID" json:"orthodontics_medical_histories,omitempty"`
}

func (p *Patient) AfterFind(_ *gorm.DB) error {
	p.FullName = p.FirstName + " " + p.LastName
	return nil
}

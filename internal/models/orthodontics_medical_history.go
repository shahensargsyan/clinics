package models

// OrthodonticsMedicalHistory is the parent record the Backpack
// OrthodonticsMedicalHistoryCrudController exposes; in the Go API this is
// the aggregation root for the "patient summary" endpoint that pulls in
// CephalometricAnalysis, DiagnosticAsset, MechanotherapyVisit, and
// ToothMeasurement via the Patient relationship.
type OrthodonticsMedicalHistory struct {
	Base
	PatientID              uint    `gorm:"not null" json:"patient_id"`
	MainComplaints         *string `gorm:"type:text" json:"main_complaints,omitempty"`
	FunctionalDisturbances *string `gorm:"type:text" json:"functional_disturbances,omitempty"`
	EntPathology           *string `gorm:"type:text" json:"ent_pathology,omitempty"`
	PosturalDisturbances   *string `gorm:"type:text" json:"postural_disturbances,omitempty"`
	BiometricFindings      *string `gorm:"type:text" json:"biometric_findings,omitempty"`
	TreatmentPlan          *string `gorm:"type:text" json:"treatment_plan,omitempty"`

	Patient *Patient `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

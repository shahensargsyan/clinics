package models

// DiagnosticAsset flags which diagnostic media the clinic has on file for
// a given patient (X-rays, CBCT scans, biometric data, photographs).
type DiagnosticAsset struct {
	Timestamped
	PatientID   uint `gorm:"not null" json:"patient_id"`
	HasOptg     bool `gorm:"not null;default:false" json:"has_optg"`
	HasTrg      bool `gorm:"not null;default:false" json:"has_trg"`
	HasCbct     bool `gorm:"not null;default:false" json:"has_cbct"`
	HasBiometry bool `gorm:"not null;default:false" json:"has_biometry"`
	HasPhotos   bool `gorm:"not null;default:false" json:"has_photos"`

	Patient *Patient `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

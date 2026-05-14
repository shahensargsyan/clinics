package models

import (
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
)

// CephalometricAnalysis captures the orthodontic measurements derived from
// a lateral cephalogram. All measurement fields are nullable — clinicians
// may record an analysis with only a subset of the angles populated.
type CephalometricAnalysis struct {
	Timestamped
	PatientID         uint             `gorm:"not null" json:"patient_id"`
	AnalysisDate      *datatypes.Date  `gorm:"type:date" json:"analysis_date,omitempty"`
	SnaDegrees        *decimal.Decimal `gorm:"type:decimal(5,2)" json:"sna_degrees,omitempty"`
	SnbDegrees        *decimal.Decimal `gorm:"type:decimal(5,2)" json:"snb_degrees,omitempty"`
	SnpogDegrees      *decimal.Decimal `gorm:"type:decimal(5,2)" json:"snpog_degrees,omitempty"`
	WitsAppraisalMm   *decimal.Decimal `gorm:"type:decimal(5,2)" json:"wits_appraisal_mm,omitempty"`
	SnPaPrime         *decimal.Decimal `gorm:"type:decimal(5,2)" json:"sn_pa_prime,omitempty"`
	NlMlDegrees       *decimal.Decimal `gorm:"type:decimal(5,2)" json:"nl_ml_degrees,omitempty"`
	BjorkSumDegrees   *decimal.Decimal `gorm:"type:decimal(5,2)" json:"bjork_sum_degrees,omitempty"`
	NGoMeDegrees      *decimal.Decimal `gorm:"type:decimal(5,2)" json:"n_go_me_degrees,omitempty"`
	GoGnLengthMm      *decimal.Decimal `gorm:"type:decimal(5,2)" json:"go_gn_length_mm,omitempty"`
	InterincisalAngle *decimal.Decimal `gorm:"type:decimal(5,2)" json:"interincisal_angle,omitempty"`
	Conclusions       *string          `gorm:"type:text" json:"conclusions,omitempty"`

	Patient *Patient `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
}

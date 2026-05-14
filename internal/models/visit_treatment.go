package models

import (
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// VisitTreatment is the pivot between Appointment and Treatment, carrying
// per-tooth quantity and pricing. The composite unique key
// (appointment_id, treatment_id, tooth_identifier) prevents accidentally
// recording the same treatment on the same tooth twice in one visit.
type VisitTreatment struct {
	Base
	UserID          *uint           `json:"user_id,omitempty"`
	AppointmentID   uint            `gorm:"not null;uniqueIndex:uk_treatment_tooth,priority:1" json:"appointment_id"`
	TreatmentID     uint            `gorm:"not null;uniqueIndex:uk_treatment_tooth,priority:2" json:"treatment_id"`
	ToothIdentifier *string         `gorm:"size:5;uniqueIndex:uk_treatment_tooth,priority:3" json:"tooth_identifier,omitempty"`
	Quantity        int             `gorm:"not null;default:1" json:"quantity"`
	ActualCost      decimal.Decimal `gorm:"type:decimal(10,2);not null" json:"actual_cost"`
	Notes           *string         `gorm:"type:text" json:"notes,omitempty"`

	// TotalCost = ActualCost * Quantity; matches the closure column the
	// Laravel VisitTreatmentCrudController rendered for the admin grid.
	TotalCost decimal.Decimal `gorm:"-" json:"total_cost"`

	User        *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Appointment *Appointment `gorm:"foreignKey:AppointmentID" json:"appointment,omitempty"`
	Treatment   *Treatment   `gorm:"foreignKey:TreatmentID" json:"treatment,omitempty"`
}

func (v *VisitTreatment) AfterFind(_ *gorm.DB) error {
	v.TotalCost = v.ActualCost.Mul(decimal.NewFromInt(int64(v.Quantity)))
	return nil
}

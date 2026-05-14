package models

import (
	"time"

	"gorm.io/gorm"
)

type AppointmentStatus string

const (
	AppointmentStatusScheduled AppointmentStatus = "Scheduled"
	AppointmentStatusConfirmed AppointmentStatus = "Confirmed"
	AppointmentStatusCompleted AppointmentStatus = "Completed"
	AppointmentStatusCancelled AppointmentStatus = "Cancelled"
)

type Appointment struct {
	Base
	PatientID           uint              `gorm:"not null" json:"patient_id"`
	UserID              uint              `gorm:"not null;uniqueIndex:uk_user_schedule,priority:1" json:"user_id"`
	AppointmentDateTime time.Time         `gorm:"not null;uniqueIndex:uk_user_schedule,priority:2" json:"appointment_date_time"`
	DurationMinutes     int               `gorm:"not null;default:30" json:"duration_minutes"`
	ReasonForVisit      *string           `gorm:"type:text" json:"reason_for_visit,omitempty"`
	Status              AppointmentStatus `gorm:"type:enum('Scheduled','Confirmed','Completed','Cancelled');not null;default:'Scheduled'" json:"status"`

	Patient         *Patient         `gorm:"foreignKey:PatientID" json:"patient,omitempty"`
	User            *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Invoice         *Invoice         `gorm:"foreignKey:AppointmentID" json:"invoice,omitempty"`
	VisitTreatments []VisitTreatment `gorm:"foreignKey:AppointmentID" json:"visit_treatments,omitempty"`
	Treatments      []Treatment      `gorm:"many2many:visit_treatments;" json:"treatments,omitempty"`
}

// Upcoming mirrors the Laravel `scopeUpcoming` query scope: future
// appointments whose status hasn't reached Completed/Cancelled yet.
// Use as: db.Scopes(models.Upcoming).Find(&appts)
func Upcoming(db *gorm.DB) *gorm.DB {
	return db.Where("appointment_date_time > ?", time.Now()).
		Where("status IN ?", []AppointmentStatus{AppointmentStatusScheduled, AppointmentStatusConfirmed})
}

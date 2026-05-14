package models

import "time"

// User is the admin/clinician account; matches Laravel's default users
// table extended with soft deletes.
type User struct {
	Base
	Name            string     `gorm:"size:255;not null" json:"name"`
	Email           string     `gorm:"size:255;not null;uniqueIndex:users_email_unique" json:"email"`
	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	Password        string     `gorm:"size:255;not null" json:"-"`
	RememberToken   *string    `gorm:"size:100" json:"-"`

	Appointments    []Appointment    `gorm:"foreignKey:UserID" json:"appointments,omitempty"`
	VisitTreatments []VisitTreatment `gorm:"foreignKey:UserID" json:"visit_treatments,omitempty"`
}

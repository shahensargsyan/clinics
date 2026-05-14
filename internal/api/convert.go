package api

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/datatypes"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

// toAPIPatient converts a GORM Patient into the API response shape. The
// model's accessor field (FullName) is expected to already be populated
// by GORM's AfterFind hook.
func toAPIPatient(p *models.Patient) Patient {
	out := Patient{
		Id:        int64(p.ID),
		FirstName: p.FirstName,
		LastName:  p.LastName,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
		Phone:     p.Phone,
		Address:   p.Address,
	}
	if p.FullName != "" {
		name := p.FullName
		out.FullName = &name
	}
	if p.DateOfBirth != nil {
		out.DateOfBirth = &openapi_types.Date{Time: time.Time(*p.DateOfBirth)}
	}
	if p.Gender != nil {
		g := PatientGender(*p.Gender)
		out.Gender = &g
	}
	if p.Email != nil {
		e := openapi_types.Email(*p.Email)
		out.Email = &e
	}
	if p.DeletedAt.Valid {
		t := p.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

func fromCreate(b PatientCreate) models.Patient {
	m := models.Patient{
		FirstName: b.FirstName,
		LastName:  b.LastName,
		Phone:     b.Phone,
		Address:   b.Address,
	}
	if b.DateOfBirth != nil {
		d := datatypes.Date(b.DateOfBirth.Time)
		m.DateOfBirth = &d
	}
	if b.Gender != nil {
		g := models.Gender(*b.Gender)
		m.Gender = &g
	}
	if b.Email != nil {
		s := string(*b.Email)
		m.Email = &s
	}
	return m
}

// toAPIAppointment converts a GORM Appointment into the response shape.
// Caller is expected to have Preload("Patient").Preload("User") when the
// nested embeds should appear; if the relations weren't loaded the embed
// fields stay nil (omitempty drops them from the payload).
func toAPIAppointment(m *models.Appointment) Appointment {
	out := Appointment{
		Id:                  int64(m.ID),
		PatientId:           int64(m.PatientID),
		UserId:              int64(m.UserID),
		AppointmentDateTime: m.AppointmentDateTime,
		DurationMinutes:     m.DurationMinutes,
		ReasonForVisit:      m.ReasonForVisit,
		Status:              AppointmentStatus(m.Status),
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
	if m.Patient != nil && m.Patient.ID != 0 {
		out.Patient = &AppointmentPatient{
			Id:       int64(m.Patient.ID),
			FullName: m.Patient.FirstName + " " + m.Patient.LastName,
		}
	}
	if m.User != nil && m.User.ID != 0 {
		email := openapi_types.Email(m.User.Email)
		out.User = &AppointmentUser{
			Id:    int64(m.User.ID),
			Name:  m.User.Name,
			Email: &email,
		}
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

func fromCreateAppointment(b AppointmentCreate) models.Appointment {
	m := models.Appointment{
		PatientID:           uint(b.PatientId),
		UserID:              uint(b.UserId),
		AppointmentDateTime: b.AppointmentDateTime,
		ReasonForVisit:      b.ReasonForVisit,
		DurationMinutes:     30,
		Status:              models.AppointmentStatusScheduled,
	}
	if b.DurationMinutes != nil {
		m.DurationMinutes = *b.DurationMinutes
	}
	if b.Status != nil {
		m.Status = models.AppointmentStatus(*b.Status)
	}
	return m
}

func applyAppointmentUpdate(m *models.Appointment, b AppointmentUpdate) {
	if b.PatientId != nil {
		m.PatientID = uint(*b.PatientId)
	}
	if b.UserId != nil {
		m.UserID = uint(*b.UserId)
	}
	if b.AppointmentDateTime != nil {
		m.AppointmentDateTime = *b.AppointmentDateTime
	}
	if b.DurationMinutes != nil {
		m.DurationMinutes = *b.DurationMinutes
	}
	if b.ReasonForVisit != nil {
		m.ReasonForVisit = b.ReasonForVisit
	}
	if b.Status != nil {
		m.Status = models.AppointmentStatus(*b.Status)
	}
}

// applyUpdate merges the request body onto an existing model. Because
// oapi-codegen's *T fields collapse "key absent" and "key: null" into the
// same nil value, this implements *partial merge* semantics: nil leaves
// the field untouched. The OpenAPI spec's "explicit null clears" promise
// is only enforceable via a custom optional/nullable type — defer until a
// real consumer needs it.
func applyUpdate(m *models.Patient, b PatientUpdate) {
	if b.FirstName != nil {
		m.FirstName = *b.FirstName
	}
	if b.LastName != nil {
		m.LastName = *b.LastName
	}
	if b.Phone != nil {
		m.Phone = b.Phone
	}
	if b.Address != nil {
		m.Address = b.Address
	}
	if b.DateOfBirth != nil {
		d := datatypes.Date(b.DateOfBirth.Time)
		m.DateOfBirth = &d
	}
	if b.Gender != nil {
		g := models.Gender(*b.Gender)
		m.Gender = &g
	}
	if b.Email != nil {
		s := string(*b.Email)
		m.Email = &s
	}
}

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

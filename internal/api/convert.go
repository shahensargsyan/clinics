package api

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/shopspring/decimal"
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

// toAPIInvoice converts a GORM Invoice into the API response shape.
// Caller should Preload("Appointment.Patient") on reads if the embedded
// appointment + patient name should appear.
func toAPIInvoice(m *models.Invoice) Invoice {
	totalStr := m.TotalAmount.StringFixed(2)
	paidStr := m.AmountPaid.StringFixed(2)
	remainingStr := m.RemainingBalance.StringFixed(2)
	out := Invoice{
		Id:               int64(m.ID),
		AppointmentId:    int64(m.AppointmentID),
		InvoiceDate:      openapi_types.Date{Time: time.Time(m.InvoiceDate)},
		TotalAmount:      totalStr,
		AmountPaid:       paidStr,
		RemainingBalance: &remainingStr,
		PaymentStatus:    InvoicePaymentStatus(m.PaymentStatus),
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
	if m.Appointment != nil && m.Appointment.ID != 0 {
		ia := InvoiceAppointment{
			Id:                  int64(m.Appointment.ID),
			AppointmentDateTime: m.Appointment.AppointmentDateTime,
		}
		if m.Appointment.Patient != nil && m.Appointment.Patient.ID != 0 {
			ia.Patient = &AppointmentPatient{
				Id:       int64(m.Appointment.Patient.ID),
				FullName: m.Appointment.Patient.FirstName + " " + m.Appointment.Patient.LastName,
			}
		}
		out.Appointment = &ia
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

// fromCreateInvoice builds a fresh model from a validated request body.
// total/paid are passed in already-parsed by the handler so this function
// stays infallible.
func fromCreateInvoice(b InvoiceCreate, total, paid decimal.Decimal) models.Invoice {
	m := models.Invoice{
		AppointmentID: uint(b.AppointmentId),
		InvoiceDate:   datatypes.Date(b.InvoiceDate.Time),
		TotalAmount:   total,
		AmountPaid:    paid,
		PaymentStatus: models.PaymentStatusPending,
	}
	if b.PaymentStatus != nil {
		m.PaymentStatus = models.PaymentStatus(*b.PaymentStatus)
	}
	return m
}

// applyInvoiceUpdate merges the partial update body. effTotal/effPaid are
// the post-merge values the handler has already cross-validated for the
// paid ≤ total invariant.
func applyInvoiceUpdate(m *models.Invoice, b InvoiceUpdate, effTotal, effPaid decimal.Decimal) {
	if b.InvoiceDate != nil {
		m.InvoiceDate = datatypes.Date(b.InvoiceDate.Time)
	}
	if b.TotalAmount != nil {
		m.TotalAmount = effTotal
	}
	if b.AmountPaid != nil {
		m.AmountPaid = effPaid
	}
	if b.PaymentStatus != nil {
		m.PaymentStatus = models.PaymentStatus(*b.PaymentStatus)
	}
}

// --- Category / Treatment / User / VisitTreatment / MedicalHistory / Clinical -----

func toAPICategory(m *models.Category) Category {
	out := Category{
		Id:        int64(m.ID),
		ParentId:  int64(m.ParentID),
		Name:      m.Name,
		Slug:      m.Slug,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

func toAPITreatment(m *models.Treatment) Treatment {
	out := Treatment{
		Id:           int64(m.ID),
		Name:         m.Name,
		Description:  m.Description,
		StandardCost: m.StandardCost.StringFixed(2),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if m.CategoryID != nil {
		cid := int64(*m.CategoryID)
		out.CategoryId = &cid
	}
	if m.Category != nil && m.Category.ID != 0 {
		cid := int64(m.Category.ID)
		out.Category = &struct {
			Id   *int64  `json:"id,omitempty"`
			Name *string `json:"name,omitempty"`
		}{Id: &cid, Name: &m.Category.Name}
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

func toAPIUser(m *models.User) User {
	out := User{
		Id:        int64(m.ID),
		Name:      m.Name,
		Email:     openapi_types.Email(m.Email),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
	if m.EmailVerifiedAt != nil {
		out.EmailVerifiedAt = m.EmailVerifiedAt
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

func toAPIVisitTreatment(m *models.VisitTreatment) VisitTreatment {
	out := VisitTreatment{
		Id:              int64(m.ID),
		AppointmentId:   int64(m.AppointmentID),
		TreatmentId:     int64(m.TreatmentID),
		Quantity:        m.Quantity,
		ActualCost:      m.ActualCost.StringFixed(2),
		ToothIdentifier: m.ToothIdentifier,
		Notes:           m.Notes,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	totalCost := m.TotalCost.StringFixed(2)
	out.TotalCost = &totalCost
	if m.UserID != nil {
		uid := int64(*m.UserID)
		out.UserId = &uid
	}
	if m.Treatment != nil && m.Treatment.ID != 0 {
		tid := int64(m.Treatment.ID)
		out.Treatment = &struct {
			Id   *int64  `json:"id,omitempty"`
			Name *string `json:"name,omitempty"`
		}{Id: &tid, Name: &m.Treatment.Name}
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

func toAPIMedicalHistory(m *models.MedicalHistory) MedicalHistory {
	out := MedicalHistory{
		Id:            int64(m.ID),
		PatientId:     int64(m.PatientID),
		ConditionName: m.ConditionName,
		Notes:         m.Notes,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

func toAPICeph(m *models.CephalometricAnalysis) CephalometricAnalysis {
	out := CephalometricAnalysis{
		Id:          int64(m.ID),
		PatientId:   int64(m.PatientID),
		Conclusions: m.Conclusions,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.AnalysisDate != nil {
		out.AnalysisDate = &openapi_types.Date{Time: time.Time(*m.AnalysisDate)}
	}
	out.SnaDegrees = decToStrPtr(m.SnaDegrees)
	out.SnbDegrees = decToStrPtr(m.SnbDegrees)
	out.SnpogDegrees = decToStrPtr(m.SnpogDegrees)
	out.WitsAppraisalMm = decToStrPtr(m.WitsAppraisalMm)
	out.SnPaPrime = decToStrPtr(m.SnPaPrime)
	out.NlMlDegrees = decToStrPtr(m.NlMlDegrees)
	out.BjorkSumDegrees = decToStrPtr(m.BjorkSumDegrees)
	out.NGoMeDegrees = decToStrPtr(m.NGoMeDegrees)
	out.GoGnLengthMm = decToStrPtr(m.GoGnLengthMm)
	out.InterincisalAngle = decToStrPtr(m.InterincisalAngle)
	return out
}

func toAPIDiag(m *models.DiagnosticAsset) DiagnosticAsset {
	return DiagnosticAsset{
		Id:          int64(m.ID),
		PatientId:   int64(m.PatientID),
		HasOptg:     m.HasOptg,
		HasTrg:      m.HasTrg,
		HasCbct:     m.HasCbct,
		HasBiometry: m.HasBiometry,
		HasPhotos:   m.HasPhotos,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func toAPIMech(m *models.MechanotherapyVisit) MechanotherapyVisit {
	out := MechanotherapyVisit{
		Id:              int64(m.ID),
		PatientId:       int64(m.PatientID),
		ProcedureNotes:  m.ProcedureNotes,
		Recommendations: m.Recommendations,
		DoctorSignature: m.DoctorSignature,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if m.VisitDate != nil {
		out.VisitDate = &openapi_types.Date{Time: time.Time(*m.VisitDate)}
	}
	return out
}

func toAPITooth(m *models.ToothMeasurement) ToothMeasurement {
	return ToothMeasurement{
		Id:                 int64(m.ID),
		PatientId:          int64(m.PatientID),
		ToothNumber:        int(m.ToothNumber),
		MesiodistalWidthMm: decToStrPtr(m.MesiodistalWidthMm),
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

func toAPIOrtho(m *models.OrthodonticsMedicalHistory) OrthodonticsMedicalHistory {
	out := OrthodonticsMedicalHistory{
		Id:                     int64(m.ID),
		PatientId:              int64(m.PatientID),
		MainComplaints:         m.MainComplaints,
		FunctionalDisturbances: m.FunctionalDisturbances,
		EntPathology:           m.EntPathology,
		PosturalDisturbances:   m.PosturalDisturbances,
		BiometricFindings:      m.BiometricFindings,
		TreatmentPlan:          m.TreatmentPlan,
		CreatedAt:              m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
	}
	if m.DeletedAt.Valid {
		t := m.DeletedAt.Time
		out.DeletedAt = &t
	}
	return out
}

// decToStrPtr formats a nullable shopspring/decimal as a *string at 2dp,
// returning nil for absent measurements. Used heavily by the cephalometric
// converter which has 10 such fields.
func decToStrPtr(d *decimal.Decimal) *string {
	if d == nil {
		return nil
	}
	s := d.StringFixed(2)
	return &s
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

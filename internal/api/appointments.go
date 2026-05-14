package api

import (
	"context"
	"errors"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

var appointmentSearchCols = []string{"reason_for_visit"}

var appointmentSortCols = map[string]struct{}{
	"id":                    {},
	"appointment_date_time": {},
	"duration_minutes":      {},
	"status":                {},
	"created_at":            {},
	"updated_at":            {},
}

func (s *Server) ListAppointments(ctx context.Context, req ListAppointmentsRequestObject) (ListAppointmentsResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)

	base := s.DB.WithContext(ctx).Model(&models.Appointment{})
	base = applySearch(base, opts.search, appointmentSearchCols)

	if p.Status != nil {
		base = base.Where("status = ?", string(*p.Status))
	}
	if p.PatientId != nil {
		base = base.Where("patient_id = ?", *p.PatientId)
	}
	if p.UserId != nil {
		base = base.Where("user_id = ?", *p.UserId)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	dataQ := base.Session(&gorm.Session{}).Preload("Patient").Preload("User")
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, appointmentSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)

	var rows []models.Appointment
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]Appointment, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIAppointment(&rows[i]))
	}
	return ListAppointments200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateAppointment(ctx context.Context, req CreateAppointmentRequestObject) (CreateAppointmentResponseObject, error) {
	if req.Body == nil {
		return apt422Create("Request body is required.", nil), nil
	}
	if errs := validateAppointmentCreate(*req.Body); len(errs) > 0 {
		return apt422Create("The given data was invalid.", errs), nil
	}
	if errs := s.validateAppointmentFKs(ctx, req.Body.PatientId, req.Body.UserId); len(errs) > 0 {
		return apt422Create("The given data was invalid.", errs), nil
	}

	m := fromCreateAppointment(*req.Body)
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		if errs := translateMySQLError(err); errs != nil {
			return apt422Create("The given data was invalid.", errs), nil
		}
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Preload("Patient").Preload("User").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return CreateAppointment201JSONResponse{Data: toAPIAppointment(&m)}, nil
}

func (s *Server) GetAppointment(ctx context.Context, req GetAppointmentRequestObject) (GetAppointmentResponseObject, error) {
	var m models.Appointment
	if err := s.DB.WithContext(ctx).Preload("Patient").Preload("User").First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetAppointment200JSONResponse{Data: toAPIAppointment(&m)}, nil
}

func (s *Server) UpdateAppointment(ctx context.Context, req UpdateAppointmentRequestObject) (UpdateAppointmentResponseObject, error) {
	if req.Body == nil {
		return apt422Update("Request body is required.", nil), nil
	}
	if errs := validateAppointmentUpdate(*req.Body); len(errs) > 0 {
		return apt422Update("The given data was invalid.", errs), nil
	}

	var m models.Appointment
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}

	if req.Body.PatientId != nil || req.Body.UserId != nil {
		// Only validate FKs that the caller is actually changing.
		pid := int64(m.PatientID)
		uid := int64(m.UserID)
		if req.Body.PatientId != nil {
			pid = *req.Body.PatientId
		}
		if req.Body.UserId != nil {
			uid = *req.Body.UserId
		}
		if errs := s.validateAppointmentFKs(ctx, pid, uid); len(errs) > 0 {
			return apt422Update("The given data was invalid.", errs), nil
		}
	}

	applyAppointmentUpdate(&m, *req.Body)
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		if errs := translateMySQLError(err); errs != nil {
			return apt422Update("The given data was invalid.", errs), nil
		}
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Preload("Patient").Preload("User").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return UpdateAppointment200JSONResponse{Data: toAPIAppointment(&m)}, nil
}

func (s *Server) DeleteAppointment(ctx context.Context, req DeleteAppointmentRequestObject) (DeleteAppointmentResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.Appointment{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteAppointment204Response{}, nil
}

// --- validation -------------------------------------------------------------

func validateAppointmentCreate(b AppointmentCreate) map[string][]string {
	errs := map[string][]string{}
	if b.PatientId <= 0 {
		errs["patient_id"] = []string{"The patient_id field is required."}
	}
	if b.UserId <= 0 {
		errs["user_id"] = []string{"The user_id field is required."}
	}
	if b.AppointmentDateTime.IsZero() {
		errs["appointment_date_time"] = []string{"The appointment_date_time field is required."}
	}
	if b.DurationMinutes != nil && (*b.DurationMinutes < 1 || *b.DurationMinutes > 600) {
		errs["duration_minutes"] = []string{"The duration_minutes must be between 1 and 600."}
	}
	if b.Status != nil && !b.Status.Valid() {
		errs["status"] = []string{"Selected status is invalid."}
	}
	return errs
}

func validateAppointmentUpdate(b AppointmentUpdate) map[string][]string {
	errs := map[string][]string{}
	if b.PatientId != nil && *b.PatientId <= 0 {
		errs["patient_id"] = []string{"The patient_id must be a positive integer."}
	}
	if b.UserId != nil && *b.UserId <= 0 {
		errs["user_id"] = []string{"The user_id must be a positive integer."}
	}
	if b.DurationMinutes != nil && (*b.DurationMinutes < 1 || *b.DurationMinutes > 600) {
		errs["duration_minutes"] = []string{"The duration_minutes must be between 1 and 600."}
	}
	if b.Status != nil && !b.Status.Valid() {
		errs["status"] = []string{"Selected status is invalid."}
	}
	return errs
}

// validateAppointmentFKs pre-checks the patient_id / user_id references so
// the API returns a structured 422 instead of bubbling a raw MySQL FK
// error from the constraint. Costs two SELECT COUNTs per write — worth
// it for the better error UX.
func (s *Server) validateAppointmentFKs(ctx context.Context, patientID, userID int64) map[string][]string {
	errs := map[string][]string{}
	var c int64
	if err := s.DB.WithContext(ctx).Model(&models.Patient{}).Where("id = ?", patientID).Count(&c).Error; err == nil && c == 0 {
		errs["patient_id"] = []string{"Patient does not exist."}
	}
	if err := s.DB.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Count(&c).Error; err == nil && c == 0 {
		errs["user_id"] = []string{"User does not exist."}
	}
	return errs
}

// translateMySQLError maps the two MySQL errors we care about
// (1062 duplicate key on uk_user_schedule, 1452 generic FK failure) into
// Laravel-shaped 422 field errors. Returns nil for everything else so the
// caller bubbles up to a 500.
func translateMySQLError(err error) map[string][]string {
	var mErr *gomysql.MySQLError
	if !errors.As(err, &mErr) {
		return nil
	}
	switch mErr.Number {
	case 1062: // duplicate entry — only one unique index on appointments
		return map[string][]string{
			"appointment_date_time": {"This clinician already has an appointment at this time."},
		}
	case 1452: // FK constraint
		return map[string][]string{
			"_": {"Referenced patient or user does not exist."},
		}
	}
	return nil
}

func apt422Create(message string, fieldErrs map[string][]string) CreateAppointment422JSONResponse {
	if fieldErrs == nil {
		fieldErrs = map[string][]string{}
	}
	return CreateAppointment422JSONResponse{ValidationErrorJSONResponse{
		Message: message,
		Errors:  fieldErrs,
	}}
}

func apt422Update(message string, fieldErrs map[string][]string) UpdateAppointment422JSONResponse {
	if fieldErrs == nil {
		fieldErrs = map[string][]string{}
	}
	return UpdateAppointment422JSONResponse{ValidationErrorJSONResponse{
		Message: message,
		Errors:  fieldErrs,
	}}
}

package api

import (
	"context"

	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

// patientSearchCols enumerates the columns the `?search=` query param
// matches against. Updating this set is the only thing required to widen
// the search surface; the SQL is built by applySearch.
var patientSearchCols = []string{"first_name", "last_name", "email", "phone"}

// patientSortCols whitelists the `?sort=` columns. MUST stay in sync with
// the OpenAPI `listPatients` description so contract and runtime agree.
var patientSortCols = map[string]struct{}{
	"id":            {},
	"first_name":    {},
	"last_name":     {},
	"date_of_birth": {},
	"email":         {},
	"created_at":    {},
	"updated_at":    {},
}

func (s *Server) ListPatients(ctx context.Context, req ListPatientsRequestObject) (ListPatientsResponseObject, error) {
	opts := normalize(req.Params.Page, req.Params.PerPage, req.Params.Search, req.Params.Sort)

	base := s.DB.WithContext(ctx).Model(&models.Patient{})
	base = applySearch(base, opts.search, patientSearchCols)

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	dataQ := base.Session(&gorm.Session{})
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, patientSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)

	var rows []models.Patient
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]Patient, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIPatient(&rows[i]))
	}
	return ListPatients200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreatePatient(ctx context.Context, req CreatePatientRequestObject) (CreatePatientResponseObject, error) {
	if req.Body == nil {
		return validation422Create("Request body is required.", nil), nil
	}
	if errs := validatePatientCreate(*req.Body); len(errs) > 0 {
		return validation422Create("The given data was invalid.", errs), nil
	}

	m := fromCreate(*req.Body)
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	// Re-fetch so AfterFind populates the FullName accessor.
	if err := s.DB.WithContext(ctx).First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return CreatePatient201JSONResponse{Data: toAPIPatient(&m)}, nil
}

func (s *Server) GetPatient(ctx context.Context, req GetPatientRequestObject) (GetPatientResponseObject, error) {
	var m models.Patient
	// gorm.ErrRecordNotFound is mapped to 404 by handlerErrorFunc.
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetPatient200JSONResponse{Data: toAPIPatient(&m)}, nil
}

func (s *Server) UpdatePatient(ctx context.Context, req UpdatePatientRequestObject) (UpdatePatientResponseObject, error) {
	if req.Body == nil {
		return validation422Update("Request body is required.", nil), nil
	}
	if errs := validatePatientUpdate(*req.Body); len(errs) > 0 {
		return validation422Update("The given data was invalid.", errs), nil
	}

	var m models.Patient
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}

	applyUpdate(&m, *req.Body)
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return UpdatePatient200JSONResponse{Data: toAPIPatient(&m)}, nil
}

func (s *Server) DeletePatient(ctx context.Context, req DeletePatientRequestObject) (DeletePatientResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.Patient{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	// GORM Delete with a missing PK silently succeeds with RowsAffected=0;
	// promote to ErrRecordNotFound so handlerErrorFunc emits a 404.
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeletePatient204Response{}, nil
}

// --- validation helpers -----------------------------------------------------

func validatePatientCreate(b PatientCreate) map[string][]string {
	errs := map[string][]string{}
	switch {
	case b.FirstName == "":
		errs["first_name"] = []string{"The first_name field is required."}
	case len(b.FirstName) > 100:
		errs["first_name"] = []string{"The first_name may not be greater than 100 characters."}
	}
	switch {
	case b.LastName == "":
		errs["last_name"] = []string{"The last_name field is required."}
	case len(b.LastName) > 100:
		errs["last_name"] = []string{"The last_name may not be greater than 100 characters."}
	}
	if b.Gender != nil && !b.Gender.Valid() {
		errs["gender"] = []string{"Selected gender is invalid."}
	}
	return errs
}

func validatePatientUpdate(b PatientUpdate) map[string][]string {
	errs := map[string][]string{}
	if b.FirstName != nil {
		switch {
		case *b.FirstName == "":
			errs["first_name"] = []string{"The first_name field cannot be empty."}
		case len(*b.FirstName) > 100:
			errs["first_name"] = []string{"The first_name may not be greater than 100 characters."}
		}
	}
	if b.LastName != nil {
		switch {
		case *b.LastName == "":
			errs["last_name"] = []string{"The last_name field cannot be empty."}
		case len(*b.LastName) > 100:
			errs["last_name"] = []string{"The last_name may not be greater than 100 characters."}
		}
	}
	if b.Gender != nil && !b.Gender.Valid() {
		errs["gender"] = []string{"Selected gender is invalid."}
	}
	return errs
}

func validation422Create(message string, fieldErrs map[string][]string) CreatePatient422JSONResponse {
	if fieldErrs == nil {
		fieldErrs = map[string][]string{}
	}
	return CreatePatient422JSONResponse{ValidationErrorJSONResponse{
		Message: message,
		Errors:  fieldErrs,
	}}
}

func validation422Update(message string, fieldErrs map[string][]string) UpdatePatient422JSONResponse {
	if fieldErrs == nil {
		fieldErrs = map[string][]string{}
	}
	return UpdatePatient422JSONResponse{ValidationErrorJSONResponse{
		Message: message,
		Errors:  fieldErrs,
	}}
}

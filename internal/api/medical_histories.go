package api

import (
	"context"

	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

var medHistSearchCols = []string{"condition_name", "notes"}
var medHistSortCols = map[string]struct{}{
	"id": {}, "condition_name": {}, "created_at": {}, "updated_at": {},
}

func (s *Server) ListMedicalHistory(ctx context.Context, req ListMedicalHistoryRequestObject) (ListMedicalHistoryResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)
	base := s.DB.WithContext(ctx).Model(&models.MedicalHistory{})
	base = applySearch(base, opts.search, medHistSearchCols)
	if p.PatientId != nil {
		base = base.Where("patient_id = ?", *p.PatientId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := base.Session(&gorm.Session{})
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, medHistSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)
	var rows []models.MedicalHistory
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]MedicalHistory, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIMedicalHistory(&rows[i]))
	}
	return ListMedicalHistory200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateMedicalHistory(ctx context.Context, req CreateMedicalHistoryRequestObject) (CreateMedicalHistoryResponseObject, error) {
	if req.Body == nil {
		return mh422Create("Request body is required.", nil), nil
	}
	errs := map[string][]string{}
	if req.Body.PatientId <= 0 {
		errs["patient_id"] = []string{"required"}
	}
	if req.Body.ConditionName == "" {
		errs["condition_name"] = []string{"required"}
	}
	if len(errs) > 0 {
		return mh422Create("The given data was invalid.", errs), nil
	}
	m := models.MedicalHistory{
		PatientID:     uint(req.Body.PatientId),
		ConditionName: req.Body.ConditionName,
		Notes:         req.Body.Notes,
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return CreateMedicalHistory201JSONResponse{Data: toAPIMedicalHistory(&m)}, nil
}

func (s *Server) GetMedicalHistory(ctx context.Context, req GetMedicalHistoryRequestObject) (GetMedicalHistoryResponseObject, error) {
	var m models.MedicalHistory
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetMedicalHistory200JSONResponse{Data: toAPIMedicalHistory(&m)}, nil
}

func (s *Server) UpdateMedicalHistory(ctx context.Context, req UpdateMedicalHistoryRequestObject) (UpdateMedicalHistoryResponseObject, error) {
	if req.Body == nil {
		return mh422Update("Request body is required.", nil), nil
	}
	var m models.MedicalHistory
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.ConditionName != nil {
		m.ConditionName = *req.Body.ConditionName
	}
	if req.Body.Notes != nil {
		m.Notes = req.Body.Notes
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return UpdateMedicalHistory200JSONResponse{Data: toAPIMedicalHistory(&m)}, nil
}

func (s *Server) DeleteMedicalHistory(ctx context.Context, req DeleteMedicalHistoryRequestObject) (DeleteMedicalHistoryResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.MedicalHistory{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteMedicalHistory204Response{}, nil
}

func mh422Create(msg string, e map[string][]string) CreateMedicalHistory422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateMedicalHistory422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}
func mh422Update(msg string, e map[string][]string) UpdateMedicalHistory422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateMedicalHistory422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}

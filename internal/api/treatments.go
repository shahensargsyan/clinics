package api

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

var treatmentSearchCols = []string{"name", "description"}
var treatmentSortCols = map[string]struct{}{
	"id": {}, "name": {}, "standard_cost": {}, "created_at": {}, "updated_at": {},
}

func (s *Server) ListTreatments(ctx context.Context, req ListTreatmentsRequestObject) (ListTreatmentsResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)
	base := s.DB.WithContext(ctx).Model(&models.Treatment{})
	base = applySearch(base, opts.search, treatmentSearchCols)
	if p.CategoryId != nil {
		base = base.Where("category_id = ?", *p.CategoryId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := base.Session(&gorm.Session{}).Preload("Category")
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, treatmentSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)
	var rows []models.Treatment
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Treatment, 0, len(rows))
	for i := range rows {
		out = append(out, toAPITreatment(&rows[i]))
	}
	return ListTreatments200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateTreatment(ctx context.Context, req CreateTreatmentRequestObject) (CreateTreatmentResponseObject, error) {
	if req.Body == nil {
		return tr422Create("Request body is required.", nil), nil
	}
	errs := map[string][]string{}
	if req.Body.Name == "" {
		errs["name"] = []string{"The name field is required."}
	}
	cost, err := decimal.NewFromString(req.Body.StandardCost)
	if err != nil || cost.IsNegative() {
		errs["standard_cost"] = []string{"Must be a non-negative decimal."}
	}
	if len(errs) > 0 {
		return tr422Create("The given data was invalid.", errs), nil
	}
	m := models.Treatment{Name: req.Body.Name, StandardCost: cost, Description: req.Body.Description}
	if req.Body.CategoryId != nil {
		id := uint(*req.Body.CategoryId)
		m.CategoryID = &id
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Preload("Category").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return CreateTreatment201JSONResponse{Data: toAPITreatment(&m)}, nil
}

func (s *Server) GetTreatment(ctx context.Context, req GetTreatmentRequestObject) (GetTreatmentResponseObject, error) {
	var m models.Treatment
	if err := s.DB.WithContext(ctx).Preload("Category").First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetTreatment200JSONResponse{Data: toAPITreatment(&m)}, nil
}

func (s *Server) UpdateTreatment(ctx context.Context, req UpdateTreatmentRequestObject) (UpdateTreatmentResponseObject, error) {
	if req.Body == nil {
		return tr422Update("Request body is required.", nil), nil
	}
	var m models.Treatment
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.Name != nil {
		m.Name = *req.Body.Name
	}
	if req.Body.Description != nil {
		m.Description = req.Body.Description
	}
	if req.Body.CategoryId != nil {
		id := uint(*req.Body.CategoryId)
		m.CategoryID = &id
	}
	if req.Body.StandardCost != nil {
		c, err := decimal.NewFromString(*req.Body.StandardCost)
		if err != nil || c.IsNegative() {
			return tr422Update("The given data was invalid.", map[string][]string{"standard_cost": {"Must be a non-negative decimal."}}), nil
		}
		m.StandardCost = c
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Preload("Category").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return UpdateTreatment200JSONResponse{Data: toAPITreatment(&m)}, nil
}

func (s *Server) DeleteTreatment(ctx context.Context, req DeleteTreatmentRequestObject) (DeleteTreatmentResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.Treatment{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteTreatment204Response{}, nil
}

func tr422Create(msg string, e map[string][]string) CreateTreatment422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateTreatment422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}
func tr422Update(msg string, e map[string][]string) UpdateTreatment422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateTreatment422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}

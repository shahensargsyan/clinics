package api

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

var visitTreatmentSearchCols = []string{"notes", "tooth_identifier"}
var visitTreatmentSortCols = map[string]struct{}{
	"id": {}, "appointment_id": {}, "treatment_id": {}, "actual_cost": {}, "quantity": {}, "created_at": {}, "updated_at": {},
}

func (s *Server) ListVisitTreatments(ctx context.Context, req ListVisitTreatmentsRequestObject) (ListVisitTreatmentsResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)
	base := s.DB.WithContext(ctx).Model(&models.VisitTreatment{})
	base = applySearch(base, opts.search, visitTreatmentSearchCols)
	if p.AppointmentId != nil {
		base = base.Where("appointment_id = ?", *p.AppointmentId)
	}
	if p.TreatmentId != nil {
		base = base.Where("treatment_id = ?", *p.TreatmentId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := base.Session(&gorm.Session{}).Preload("Treatment")
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, visitTreatmentSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)
	var rows []models.VisitTreatment
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]VisitTreatment, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIVisitTreatment(&rows[i]))
	}
	return ListVisitTreatments200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateVisitTreatment(ctx context.Context, req CreateVisitTreatmentRequestObject) (CreateVisitTreatmentResponseObject, error) {
	if req.Body == nil {
		return vt422Create("Request body is required.", nil), nil
	}
	errs := map[string][]string{}
	if req.Body.AppointmentId <= 0 {
		errs["appointment_id"] = []string{"required"}
	}
	if req.Body.TreatmentId <= 0 {
		errs["treatment_id"] = []string{"required"}
	}
	cost, err := decimal.NewFromString(req.Body.ActualCost)
	if err != nil || cost.IsNegative() {
		errs["actual_cost"] = []string{"Must be a non-negative decimal."}
	}
	if len(errs) > 0 {
		return vt422Create("The given data was invalid.", errs), nil
	}
	m := models.VisitTreatment{
		AppointmentID:   uint(req.Body.AppointmentId),
		TreatmentID:     uint(req.Body.TreatmentId),
		ActualCost:      cost,
		Quantity:        1,
		ToothIdentifier: req.Body.ToothIdentifier,
		Notes:           req.Body.Notes,
	}
	if req.Body.Quantity != nil {
		m.Quantity = *req.Body.Quantity
	}
	if req.Body.UserId != nil {
		u := uint(*req.Body.UserId)
		m.UserID = &u
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		if isMySQLDuplicate(err) {
			return vt422Create("The given data was invalid.", map[string][]string{"tooth_identifier": {"This treatment+tooth is already recorded for the appointment."}}), nil
		}
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Preload("Treatment").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return CreateVisitTreatment201JSONResponse{Data: toAPIVisitTreatment(&m)}, nil
}

func (s *Server) GetVisitTreatment(ctx context.Context, req GetVisitTreatmentRequestObject) (GetVisitTreatmentResponseObject, error) {
	var m models.VisitTreatment
	if err := s.DB.WithContext(ctx).Preload("Treatment").First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetVisitTreatment200JSONResponse{Data: toAPIVisitTreatment(&m)}, nil
}

func (s *Server) UpdateVisitTreatment(ctx context.Context, req UpdateVisitTreatmentRequestObject) (UpdateVisitTreatmentResponseObject, error) {
	if req.Body == nil {
		return vt422Update("Request body is required.", nil), nil
	}
	var m models.VisitTreatment
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.UserId != nil {
		u := uint(*req.Body.UserId)
		m.UserID = &u
	}
	if req.Body.ToothIdentifier != nil {
		m.ToothIdentifier = req.Body.ToothIdentifier
	}
	if req.Body.Quantity != nil {
		m.Quantity = *req.Body.Quantity
	}
	if req.Body.Notes != nil {
		m.Notes = req.Body.Notes
	}
	if req.Body.ActualCost != nil {
		c, err := decimal.NewFromString(*req.Body.ActualCost)
		if err != nil || c.IsNegative() {
			return vt422Update("The given data was invalid.", map[string][]string{"actual_cost": {"Must be a non-negative decimal."}}), nil
		}
		m.ActualCost = c
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Preload("Treatment").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return UpdateVisitTreatment200JSONResponse{Data: toAPIVisitTreatment(&m)}, nil
}

func (s *Server) DeleteVisitTreatment(ctx context.Context, req DeleteVisitTreatmentRequestObject) (DeleteVisitTreatmentResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.VisitTreatment{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteVisitTreatment204Response{}, nil
}

func vt422Create(msg string, e map[string][]string) CreateVisitTreatment422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateVisitTreatment422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}
func vt422Update(msg string, e map[string][]string) UpdateVisitTreatment422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateVisitTreatment422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}

package api

import (
	"context"

	"github.com/gosimple/slug"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

var categorySearchCols = []string{"name", "slug"}
var categorySortCols = map[string]struct{}{
	"id": {}, "name": {}, "slug": {}, "created_at": {}, "updated_at": {},
}

func (s *Server) ListCategories(ctx context.Context, req ListCategoriesRequestObject) (ListCategoriesResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)

	base := s.DB.WithContext(ctx).Model(&models.Category{})
	base = applySearch(base, opts.search, categorySearchCols)
	if p.ParentId != nil {
		base = base.Where("parent_id = ?", *p.ParentId)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := base.Session(&gorm.Session{})
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, categorySortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)

	var rows []models.Category
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]Category, 0, len(rows))
	for i := range rows {
		out = append(out, toAPICategory(&rows[i]))
	}
	return ListCategories200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateCategory(ctx context.Context, req CreateCategoryRequestObject) (CreateCategoryResponseObject, error) {
	if req.Body == nil {
		return cat422Create("Request body is required.", nil), nil
	}
	if req.Body.Name == "" {
		return cat422Create("The given data was invalid.", map[string][]string{"name": {"The name field is required."}}), nil
	}
	m := models.Category{Name: req.Body.Name}
	if req.Body.ParentId != nil {
		m.ParentID = uint(*req.Body.ParentId)
	}
	if req.Body.Slug != nil && *req.Body.Slug != "" {
		m.Slug = *req.Body.Slug
	} else {
		m.Slug = slug.Make(req.Body.Name)
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return CreateCategory201JSONResponse{Data: toAPICategory(&m)}, nil
}

func (s *Server) GetCategory(ctx context.Context, req GetCategoryRequestObject) (GetCategoryResponseObject, error) {
	var m models.Category
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetCategory200JSONResponse{Data: toAPICategory(&m)}, nil
}

func (s *Server) UpdateCategory(ctx context.Context, req UpdateCategoryRequestObject) (UpdateCategoryResponseObject, error) {
	if req.Body == nil {
		return cat422Update("Request body is required.", nil), nil
	}
	var m models.Category
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.Name != nil {
		m.Name = *req.Body.Name
	}
	if req.Body.ParentId != nil {
		m.ParentID = uint(*req.Body.ParentId)
	}
	if req.Body.Slug != nil {
		m.Slug = *req.Body.Slug
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return UpdateCategory200JSONResponse{Data: toAPICategory(&m)}, nil
}

func (s *Server) DeleteCategory(ctx context.Context, req DeleteCategoryRequestObject) (DeleteCategoryResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.Category{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteCategory204Response{}, nil
}

func cat422Create(msg string, e map[string][]string) CreateCategory422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateCategory422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}
func cat422Update(msg string, e map[string][]string) UpdateCategory422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateCategory422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}

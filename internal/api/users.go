package api

import (
	"context"
	"errors"

	gomysql "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

var userSearchCols = []string{"name", "email"}
var userSortCols = map[string]struct{}{
	"id": {}, "name": {}, "email": {}, "created_at": {}, "updated_at": {},
}

func (s *Server) ListUsers(ctx context.Context, req ListUsersRequestObject) (ListUsersResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)
	base := s.DB.WithContext(ctx).Model(&models.User{})
	base = applySearch(base, opts.search, userSearchCols)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := base.Session(&gorm.Session{})
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, userSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)
	var rows []models.User
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]User, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIUser(&rows[i]))
	}
	return ListUsers200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateUser(ctx context.Context, req CreateUserRequestObject) (CreateUserResponseObject, error) {
	if req.Body == nil {
		return usr422Create("Request body is required.", nil), nil
	}
	errs := map[string][]string{}
	if req.Body.Name == "" {
		errs["name"] = []string{"The name field is required."}
	}
	if req.Body.Email == "" {
		errs["email"] = []string{"The email field is required."}
	}
	if len(req.Body.Password) < 8 {
		errs["password"] = []string{"The password must be at least 8 characters."}
	}
	if len(errs) > 0 {
		return usr422Create("The given data was invalid.", errs), nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Body.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	m := models.User{
		Name:     req.Body.Name,
		Email:    string(req.Body.Email),
		Password: string(hash),
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		if isMySQLDuplicate(err) {
			return usr422Create("The given data was invalid.", map[string][]string{"email": {"Email is already taken."}}), nil
		}
		return nil, err
	}
	return CreateUser201JSONResponse{Data: toAPIUser(&m)}, nil
}

func (s *Server) GetUser(ctx context.Context, req GetUserRequestObject) (GetUserResponseObject, error) {
	var m models.User
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetUser200JSONResponse{Data: toAPIUser(&m)}, nil
}

func (s *Server) UpdateUser(ctx context.Context, req UpdateUserRequestObject) (UpdateUserResponseObject, error) {
	if req.Body == nil {
		return usr422Update("Request body is required.", nil), nil
	}
	var m models.User
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.Name != nil {
		m.Name = *req.Body.Name
	}
	if req.Body.Email != nil {
		m.Email = string(*req.Body.Email)
	}
	if req.Body.Password != nil {
		if len(*req.Body.Password) < 8 {
			return usr422Update("The given data was invalid.", map[string][]string{"password": {"The password must be at least 8 characters."}}), nil
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Body.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		m.Password = string(hash)
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		if isMySQLDuplicate(err) {
			return usr422Update("The given data was invalid.", map[string][]string{"email": {"Email is already taken."}}), nil
		}
		return nil, err
	}
	return UpdateUser200JSONResponse{Data: toAPIUser(&m)}, nil
}

func (s *Server) DeleteUser(ctx context.Context, req DeleteUserRequestObject) (DeleteUserResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.User{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteUser204Response{}, nil
}

func isMySQLDuplicate(err error) bool {
	var mErr *gomysql.MySQLError
	return errors.As(err, &mErr) && mErr.Number == 1062
}

func usr422Create(msg string, e map[string][]string) CreateUser422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateUser422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}
func usr422Update(msg string, e map[string][]string) UpdateUser422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateUser422JSONResponse{ValidationErrorJSONResponse{Message: msg, Errors: e}}
}

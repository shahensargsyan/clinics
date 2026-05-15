// Package api: clinical.go bundles the five orthodontic specialty
// resources whose handlers all follow the same skinny pattern —
// patient-id-keyed, no embedded relations beyond the patient itself,
// mostly decimal fields. Lumped into one file to avoid five
// near-identical 100-line files.
package api

import (
	"context"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

// --- CephalometricAnalysis --------------------------------------------------

var cephSortCols = map[string]struct{}{"id": {}, "analysis_date": {}, "created_at": {}, "updated_at": {}}

func (s *Server) ListCephalometricAnalyses(ctx context.Context, req ListCephalometricAnalysesRequestObject) (ListCephalometricAnalysesResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, nil, p.Sort)
	base := s.DB.WithContext(ctx).Model(&models.CephalometricAnalysis{})
	if p.PatientId != nil {
		base = base.Where("patient_id = ?", *p.PatientId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := base.Session(&gorm.Session{})
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, cephSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)
	var rows []models.CephalometricAnalysis
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]CephalometricAnalysis, 0, len(rows))
	for i := range rows {
		out = append(out, toAPICeph(&rows[i]))
	}
	return ListCephalometricAnalyses200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateCephalometricAnalysis(ctx context.Context, req CreateCephalometricAnalysisRequestObject) (CreateCephalometricAnalysisResponseObject, error) {
	if req.Body == nil || req.Body.PatientId <= 0 {
		return ceph422Create("The given data was invalid.", map[string][]string{"patient_id": {"required"}}), nil
	}
	m := models.CephalometricAnalysis{PatientID: uint(req.Body.PatientId), Conclusions: req.Body.Conclusions}
	if req.Body.AnalysisDate != nil {
		d := datatypes.Date(req.Body.AnalysisDate.Time)
		m.AnalysisDate = &d
	}
	applyCephDecimals(&m, req.Body.SnaDegrees, req.Body.SnbDegrees, req.Body.SnpogDegrees, req.Body.WitsAppraisalMm,
		req.Body.SnPaPrime, req.Body.NlMlDegrees, req.Body.BjorkSumDegrees, req.Body.NGoMeDegrees,
		req.Body.GoGnLengthMm, req.Body.InterincisalAngle)
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return CreateCephalometricAnalysis201JSONResponse{Data: toAPICeph(&m)}, nil
}

func (s *Server) GetCephalometricAnalysis(ctx context.Context, req GetCephalometricAnalysisRequestObject) (GetCephalometricAnalysisResponseObject, error) {
	var m models.CephalometricAnalysis
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetCephalometricAnalysis200JSONResponse{Data: toAPICeph(&m)}, nil
}

func (s *Server) UpdateCephalometricAnalysis(ctx context.Context, req UpdateCephalometricAnalysisRequestObject) (UpdateCephalometricAnalysisResponseObject, error) {
	if req.Body == nil {
		return ceph422Update("Request body is required.", nil), nil
	}
	var m models.CephalometricAnalysis
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.AnalysisDate != nil {
		d := datatypes.Date(req.Body.AnalysisDate.Time)
		m.AnalysisDate = &d
	}
	if req.Body.Conclusions != nil {
		m.Conclusions = req.Body.Conclusions
	}
	applyCephDecimals(&m, req.Body.SnaDegrees, req.Body.SnbDegrees, req.Body.SnpogDegrees, req.Body.WitsAppraisalMm,
		req.Body.SnPaPrime, req.Body.NlMlDegrees, req.Body.BjorkSumDegrees, req.Body.NGoMeDegrees,
		req.Body.GoGnLengthMm, req.Body.InterincisalAngle)
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return UpdateCephalometricAnalysis200JSONResponse{Data: toAPICeph(&m)}, nil
}

func (s *Server) DeleteCephalometricAnalysis(ctx context.Context, req DeleteCephalometricAnalysisRequestObject) (DeleteCephalometricAnalysisResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.CephalometricAnalysis{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteCephalometricAnalysis204Response{}, nil
}

// --- DiagnosticAsset --------------------------------------------------------

func (s *Server) ListDiagnosticAssets(ctx context.Context, req ListDiagnosticAssetsRequestObject) (ListDiagnosticAssetsResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, nil, nil)
	base := s.DB.WithContext(ctx).Model(&models.DiagnosticAsset{})
	if p.PatientId != nil {
		base = base.Where("patient_id = ?", *p.PatientId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ, meta := applyPaginate(base.Session(&gorm.Session{}).Order("id DESC"), opts.page, opts.perPage, total)
	var rows []models.DiagnosticAsset
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]DiagnosticAsset, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIDiag(&rows[i]))
	}
	return ListDiagnosticAssets200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateDiagnosticAsset(ctx context.Context, req CreateDiagnosticAssetRequestObject) (CreateDiagnosticAssetResponseObject, error) {
	if req.Body == nil || req.Body.PatientId <= 0 {
		return diag422Create("The given data was invalid.", map[string][]string{"patient_id": {"required"}}), nil
	}
	m := models.DiagnosticAsset{PatientID: uint(req.Body.PatientId)}
	if req.Body.HasOptg != nil {
		m.HasOptg = *req.Body.HasOptg
	}
	if req.Body.HasTrg != nil {
		m.HasTrg = *req.Body.HasTrg
	}
	if req.Body.HasCbct != nil {
		m.HasCbct = *req.Body.HasCbct
	}
	if req.Body.HasBiometry != nil {
		m.HasBiometry = *req.Body.HasBiometry
	}
	if req.Body.HasPhotos != nil {
		m.HasPhotos = *req.Body.HasPhotos
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return CreateDiagnosticAsset201JSONResponse{Data: toAPIDiag(&m)}, nil
}

func (s *Server) GetDiagnosticAsset(ctx context.Context, req GetDiagnosticAssetRequestObject) (GetDiagnosticAssetResponseObject, error) {
	var m models.DiagnosticAsset
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetDiagnosticAsset200JSONResponse{Data: toAPIDiag(&m)}, nil
}

func (s *Server) UpdateDiagnosticAsset(ctx context.Context, req UpdateDiagnosticAssetRequestObject) (UpdateDiagnosticAssetResponseObject, error) {
	if req.Body == nil {
		return diag422Update("Request body is required.", nil), nil
	}
	var m models.DiagnosticAsset
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.HasOptg != nil {
		m.HasOptg = *req.Body.HasOptg
	}
	if req.Body.HasTrg != nil {
		m.HasTrg = *req.Body.HasTrg
	}
	if req.Body.HasCbct != nil {
		m.HasCbct = *req.Body.HasCbct
	}
	if req.Body.HasBiometry != nil {
		m.HasBiometry = *req.Body.HasBiometry
	}
	if req.Body.HasPhotos != nil {
		m.HasPhotos = *req.Body.HasPhotos
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return UpdateDiagnosticAsset200JSONResponse{Data: toAPIDiag(&m)}, nil
}

func (s *Server) DeleteDiagnosticAsset(ctx context.Context, req DeleteDiagnosticAssetRequestObject) (DeleteDiagnosticAssetResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.DiagnosticAsset{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteDiagnosticAsset204Response{}, nil
}

// --- MechanotherapyVisit ----------------------------------------------------

var mechSearchCols = []string{"procedure_notes", "recommendations", "doctor_signature"}
var mechSortCols = map[string]struct{}{"id": {}, "visit_date": {}, "created_at": {}, "updated_at": {}}

func (s *Server) ListMechanotherapyVisits(ctx context.Context, req ListMechanotherapyVisitsRequestObject) (ListMechanotherapyVisitsResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)
	base := s.DB.WithContext(ctx).Model(&models.MechanotherapyVisit{})
	base = applySearch(base, opts.search, mechSearchCols)
	if p.PatientId != nil {
		base = base.Where("patient_id = ?", *p.PatientId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := applySort(base.Session(&gorm.Session{}), opts.sortCol, opts.sortDir, mechSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)
	var rows []models.MechanotherapyVisit
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]MechanotherapyVisit, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIMech(&rows[i]))
	}
	return ListMechanotherapyVisits200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateMechanotherapyVisit(ctx context.Context, req CreateMechanotherapyVisitRequestObject) (CreateMechanotherapyVisitResponseObject, error) {
	if req.Body == nil || req.Body.PatientId <= 0 {
		return mech422Create("The given data was invalid.", map[string][]string{"patient_id": {"required"}}), nil
	}
	m := models.MechanotherapyVisit{
		PatientID:       uint(req.Body.PatientId),
		ProcedureNotes:  req.Body.ProcedureNotes,
		Recommendations: req.Body.Recommendations,
		DoctorSignature: req.Body.DoctorSignature,
	}
	if req.Body.VisitDate != nil {
		d := datatypes.Date(req.Body.VisitDate.Time)
		m.VisitDate = &d
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return CreateMechanotherapyVisit201JSONResponse{Data: toAPIMech(&m)}, nil
}

func (s *Server) GetMechanotherapyVisit(ctx context.Context, req GetMechanotherapyVisitRequestObject) (GetMechanotherapyVisitResponseObject, error) {
	var m models.MechanotherapyVisit
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetMechanotherapyVisit200JSONResponse{Data: toAPIMech(&m)}, nil
}

func (s *Server) UpdateMechanotherapyVisit(ctx context.Context, req UpdateMechanotherapyVisitRequestObject) (UpdateMechanotherapyVisitResponseObject, error) {
	if req.Body == nil {
		return mech422Update("Request body is required.", nil), nil
	}
	var m models.MechanotherapyVisit
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.VisitDate != nil {
		d := datatypes.Date(req.Body.VisitDate.Time)
		m.VisitDate = &d
	}
	if req.Body.ProcedureNotes != nil {
		m.ProcedureNotes = req.Body.ProcedureNotes
	}
	if req.Body.Recommendations != nil {
		m.Recommendations = req.Body.Recommendations
	}
	if req.Body.DoctorSignature != nil {
		m.DoctorSignature = req.Body.DoctorSignature
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return UpdateMechanotherapyVisit200JSONResponse{Data: toAPIMech(&m)}, nil
}

func (s *Server) DeleteMechanotherapyVisit(ctx context.Context, req DeleteMechanotherapyVisitRequestObject) (DeleteMechanotherapyVisitResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.MechanotherapyVisit{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteMechanotherapyVisit204Response{}, nil
}

// --- ToothMeasurement -------------------------------------------------------

func (s *Server) ListToothMeasurements(ctx context.Context, req ListToothMeasurementsRequestObject) (ListToothMeasurementsResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, nil, nil)
	base := s.DB.WithContext(ctx).Model(&models.ToothMeasurement{})
	if p.PatientId != nil {
		base = base.Where("patient_id = ?", *p.PatientId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ, meta := applyPaginate(base.Session(&gorm.Session{}).Order("tooth_number ASC"), opts.page, opts.perPage, total)
	var rows []models.ToothMeasurement
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ToothMeasurement, 0, len(rows))
	for i := range rows {
		out = append(out, toAPITooth(&rows[i]))
	}
	return ListToothMeasurements200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateToothMeasurement(ctx context.Context, req CreateToothMeasurementRequestObject) (CreateToothMeasurementResponseObject, error) {
	if req.Body == nil || req.Body.PatientId <= 0 {
		return tooth422Create("The given data was invalid.", map[string][]string{"patient_id": {"required"}}), nil
	}
	m := models.ToothMeasurement{
		PatientID:   uint(req.Body.PatientId),
		ToothNumber: int8(req.Body.ToothNumber),
	}
	if req.Body.MesiodistalWidthMm != nil && *req.Body.MesiodistalWidthMm != "" {
		v, err := decimal.NewFromString(*req.Body.MesiodistalWidthMm)
		if err != nil {
			return tooth422Create("The given data was invalid.", map[string][]string{"mesiodistal_width_mm": {"Must be a decimal."}}), nil
		}
		m.MesiodistalWidthMm = &v
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return CreateToothMeasurement201JSONResponse{Data: toAPITooth(&m)}, nil
}

func (s *Server) GetToothMeasurement(ctx context.Context, req GetToothMeasurementRequestObject) (GetToothMeasurementResponseObject, error) {
	var m models.ToothMeasurement
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetToothMeasurement200JSONResponse{Data: toAPITooth(&m)}, nil
}

func (s *Server) UpdateToothMeasurement(ctx context.Context, req UpdateToothMeasurementRequestObject) (UpdateToothMeasurementResponseObject, error) {
	if req.Body == nil {
		return tooth422Update("Request body is required.", nil), nil
	}
	var m models.ToothMeasurement
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.ToothNumber != nil {
		m.ToothNumber = int8(*req.Body.ToothNumber)
	}
	if req.Body.MesiodistalWidthMm != nil {
		if *req.Body.MesiodistalWidthMm == "" {
			m.MesiodistalWidthMm = nil
		} else {
			v, err := decimal.NewFromString(*req.Body.MesiodistalWidthMm)
			if err != nil {
				return tooth422Update("The given data was invalid.", map[string][]string{"mesiodistal_width_mm": {"Must be a decimal."}}), nil
			}
			m.MesiodistalWidthMm = &v
		}
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return UpdateToothMeasurement200JSONResponse{Data: toAPITooth(&m)}, nil
}

func (s *Server) DeleteToothMeasurement(ctx context.Context, req DeleteToothMeasurementRequestObject) (DeleteToothMeasurementResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.ToothMeasurement{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteToothMeasurement204Response{}, nil
}

// --- OrthodonticsMedicalHistory ---------------------------------------------

var orthoSearchCols = []string{"main_complaints", "functional_disturbances", "ent_pathology", "postural_disturbances", "biometric_findings", "treatment_plan"}
var orthoSortCols = map[string]struct{}{"id": {}, "created_at": {}, "updated_at": {}}

func (s *Server) ListOrthodonticsMedicalHistories(ctx context.Context, req ListOrthodonticsMedicalHistoriesRequestObject) (ListOrthodonticsMedicalHistoriesResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)
	base := s.DB.WithContext(ctx).Model(&models.OrthodonticsMedicalHistory{})
	base = applySearch(base, opts.search, orthoSearchCols)
	if p.PatientId != nil {
		base = base.Where("patient_id = ?", *p.PatientId)
	}
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	dataQ := applySort(base.Session(&gorm.Session{}), opts.sortCol, opts.sortDir, orthoSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)
	var rows []models.OrthodonticsMedicalHistory
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]OrthodonticsMedicalHistory, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIOrtho(&rows[i]))
	}
	return ListOrthodonticsMedicalHistories200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateOrthodonticsMedicalHistory(ctx context.Context, req CreateOrthodonticsMedicalHistoryRequestObject) (CreateOrthodonticsMedicalHistoryResponseObject, error) {
	if req.Body == nil || req.Body.PatientId <= 0 {
		return ortho422Create("The given data was invalid.", map[string][]string{"patient_id": {"required"}}), nil
	}
	m := models.OrthodonticsMedicalHistory{
		PatientID:              uint(req.Body.PatientId),
		MainComplaints:         req.Body.MainComplaints,
		FunctionalDisturbances: req.Body.FunctionalDisturbances,
		EntPathology:           req.Body.EntPathology,
		PosturalDisturbances:   req.Body.PosturalDisturbances,
		BiometricFindings:      req.Body.BiometricFindings,
		TreatmentPlan:          req.Body.TreatmentPlan,
	}
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	return CreateOrthodonticsMedicalHistory201JSONResponse{Data: toAPIOrtho(&m)}, nil
}

func (s *Server) GetOrthodonticsMedicalHistory(ctx context.Context, req GetOrthodonticsMedicalHistoryRequestObject) (GetOrthodonticsMedicalHistoryResponseObject, error) {
	var m models.OrthodonticsMedicalHistory
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetOrthodonticsMedicalHistory200JSONResponse{Data: toAPIOrtho(&m)}, nil
}

func (s *Server) UpdateOrthodonticsMedicalHistory(ctx context.Context, req UpdateOrthodonticsMedicalHistoryRequestObject) (UpdateOrthodonticsMedicalHistoryResponseObject, error) {
	if req.Body == nil {
		return ortho422Update("Request body is required.", nil), nil
	}
	var m models.OrthodonticsMedicalHistory
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	if req.Body.MainComplaints != nil {
		m.MainComplaints = req.Body.MainComplaints
	}
	if req.Body.FunctionalDisturbances != nil {
		m.FunctionalDisturbances = req.Body.FunctionalDisturbances
	}
	if req.Body.EntPathology != nil {
		m.EntPathology = req.Body.EntPathology
	}
	if req.Body.PosturalDisturbances != nil {
		m.PosturalDisturbances = req.Body.PosturalDisturbances
	}
	if req.Body.BiometricFindings != nil {
		m.BiometricFindings = req.Body.BiometricFindings
	}
	if req.Body.TreatmentPlan != nil {
		m.TreatmentPlan = req.Body.TreatmentPlan
	}
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		return nil, err
	}
	return UpdateOrthodonticsMedicalHistory200JSONResponse{Data: toAPIOrtho(&m)}, nil
}

func (s *Server) DeleteOrthodonticsMedicalHistory(ctx context.Context, req DeleteOrthodonticsMedicalHistoryRequestObject) (DeleteOrthodonticsMedicalHistoryResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.OrthodonticsMedicalHistory{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteOrthodonticsMedicalHistory204Response{}, nil
}

// --- 422 response helpers ---------------------------------------------------

func ceph422Create(m string, e map[string][]string) CreateCephalometricAnalysis422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateCephalometricAnalysis422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func ceph422Update(m string, e map[string][]string) UpdateCephalometricAnalysis422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateCephalometricAnalysis422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func diag422Create(m string, e map[string][]string) CreateDiagnosticAsset422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateDiagnosticAsset422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func diag422Update(m string, e map[string][]string) UpdateDiagnosticAsset422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateDiagnosticAsset422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func mech422Create(m string, e map[string][]string) CreateMechanotherapyVisit422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateMechanotherapyVisit422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func mech422Update(m string, e map[string][]string) UpdateMechanotherapyVisit422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateMechanotherapyVisit422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func tooth422Create(m string, e map[string][]string) CreateToothMeasurement422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateToothMeasurement422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func tooth422Update(m string, e map[string][]string) UpdateToothMeasurement422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateToothMeasurement422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func ortho422Create(m string, e map[string][]string) CreateOrthodonticsMedicalHistory422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return CreateOrthodonticsMedicalHistory422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}
func ortho422Update(m string, e map[string][]string) UpdateOrthodonticsMedicalHistory422JSONResponse {
	if e == nil {
		e = map[string][]string{}
	}
	return UpdateOrthodonticsMedicalHistory422JSONResponse{ValidationErrorJSONResponse{Message: m, Errors: e}}
}

// applyCephDecimals folds the 10 nullable decimal-string fields onto the
// CephalometricAnalysis model in one place. Unparseable strings silently
// stay at their existing value — these are scientific measurements that
// should already have been validated client-side; the API tolerates noise
// rather than rejecting a partial entry.
func applyCephDecimals(m *models.CephalometricAnalysis, sna, snb, snpog, wits, snPa, nlMl, bjork, nGoMe, goGn, inter *string) {
	setIfDecimal := func(dst **decimal.Decimal, src *string) {
		if src == nil {
			return
		}
		if *src == "" {
			*dst = nil
			return
		}
		v, err := decimal.NewFromString(*src)
		if err == nil {
			*dst = &v
		}
	}
	setIfDecimal(&m.SnaDegrees, sna)
	setIfDecimal(&m.SnbDegrees, snb)
	setIfDecimal(&m.SnpogDegrees, snpog)
	setIfDecimal(&m.WitsAppraisalMm, wits)
	setIfDecimal(&m.SnPaPrime, snPa)
	setIfDecimal(&m.NlMlDegrees, nlMl)
	setIfDecimal(&m.BjorkSumDegrees, bjork)
	setIfDecimal(&m.NGoMeDegrees, nGoMe)
	setIfDecimal(&m.GoGnLengthMm, goGn)
	setIfDecimal(&m.InterincisalAngle, inter)
}

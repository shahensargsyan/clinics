package api

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

// GetPatientSummary aggregates every clinical sub-record for one patient
// in a single response — replaces the Laravel
// OrthodonticsMedicalHistoryCrudController rollup. Fetches in parallel-
// safe order; if the patient itself doesn't exist returns 404 via the
// gorm.ErrRecordNotFound funnel, otherwise empty arrays are returned for
// sub-types the patient has no records for.
func (s *Server) GetPatientSummary(ctx context.Context, req GetPatientSummaryRequestObject) (GetPatientSummaryResponseObject, error) {
	db := s.DB.WithContext(ctx)
	pid := req.Id

	var patient models.Patient
	if err := db.First(&patient, pid).Error; err != nil {
		return nil, err
	}

	out := PatientSummary{PatientId: pid}

	var mh []models.MedicalHistory
	if err := db.Where("patient_id = ?", pid).Order("id DESC").Find(&mh).Error; err != nil {
		return nil, err
	}
	mhOut := make([]MedicalHistory, 0, len(mh))
	for i := range mh {
		mhOut = append(mhOut, toAPIMedicalHistory(&mh[i]))
	}
	out.MedicalHistory = &mhOut

	var ortho models.OrthodonticsMedicalHistory
	if err := db.Where("patient_id = ?", pid).Order("id DESC").First(&ortho).Error; err == nil {
		ov := toAPIOrtho(&ortho)
		out.OrthodonticsMedicalHistory = &ov
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var ceph []models.CephalometricAnalysis
	if err := db.Where("patient_id = ?", pid).Order("id DESC").Find(&ceph).Error; err != nil {
		return nil, err
	}
	cephOut := make([]CephalometricAnalysis, 0, len(ceph))
	for i := range ceph {
		cephOut = append(cephOut, toAPICeph(&ceph[i]))
	}
	out.CephalometricAnalyses = &cephOut

	var diag []models.DiagnosticAsset
	if err := db.Where("patient_id = ?", pid).Order("id DESC").Find(&diag).Error; err != nil {
		return nil, err
	}
	diagOut := make([]DiagnosticAsset, 0, len(diag))
	for i := range diag {
		diagOut = append(diagOut, toAPIDiag(&diag[i]))
	}
	out.DiagnosticAssets = &diagOut

	var mech []models.MechanotherapyVisit
	if err := db.Where("patient_id = ?", pid).Order("visit_date DESC NULLS LAST, id DESC").Find(&mech).Error; err != nil {
		// MySQL doesn't support NULLS LAST natively before 8.0; fall back to a simpler order on failure.
		if err := db.Where("patient_id = ?", pid).Order("id DESC").Find(&mech).Error; err != nil {
			return nil, err
		}
	}
	mechOut := make([]MechanotherapyVisit, 0, len(mech))
	for i := range mech {
		mechOut = append(mechOut, toAPIMech(&mech[i]))
	}
	out.MechanotherapyVisits = &mechOut

	var teeth []models.ToothMeasurement
	if err := db.Where("patient_id = ?", pid).Order("tooth_number ASC").Find(&teeth).Error; err != nil {
		return nil, err
	}
	teethOut := make([]ToothMeasurement, 0, len(teeth))
	for i := range teeth {
		teethOut = append(teethOut, toAPITooth(&teeth[i]))
	}
	out.ToothMeasurements = &teethOut

	return GetPatientSummary200JSONResponse(out), nil
}

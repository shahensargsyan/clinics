package api

import (
	"context"
	"errors"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/shahensargsyan/my-new-go-api/internal/models"
)

// invoiceSearchCols is intentionally empty — there is no free-text column
// on invoices worth searching. ?search= is documented as a no-op for this
// resource.
var invoiceSearchCols = []string{}

var invoiceSortCols = map[string]struct{}{
	"id":             {},
	"invoice_date":   {},
	"total_amount":   {},
	"amount_paid":    {},
	"payment_status": {},
	"created_at":     {},
	"updated_at":     {},
}

func (s *Server) ListInvoices(ctx context.Context, req ListInvoicesRequestObject) (ListInvoicesResponseObject, error) {
	p := req.Params
	opts := normalize(p.Page, p.PerPage, p.Search, p.Sort)

	base := s.DB.WithContext(ctx).Model(&models.Invoice{})
	base = applySearch(base, opts.search, invoiceSearchCols)

	if p.PaymentStatus != nil {
		base = base.Where("payment_status = ?", string(*p.PaymentStatus))
	}
	if p.AppointmentId != nil {
		base = base.Where("appointment_id = ?", *p.AppointmentId)
	}

	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}

	dataQ := base.Session(&gorm.Session{}).Preload("Appointment.Patient")
	dataQ = applySort(dataQ, opts.sortCol, opts.sortDir, invoiceSortCols)
	dataQ, meta := applyPaginate(dataQ, opts.page, opts.perPage, total)

	var rows []models.Invoice
	if err := dataQ.Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]Invoice, 0, len(rows))
	for i := range rows {
		out = append(out, toAPIInvoice(&rows[i]))
	}
	return ListInvoices200JSONResponse{Data: out, Meta: meta}, nil
}

func (s *Server) CreateInvoice(ctx context.Context, req CreateInvoiceRequestObject) (CreateInvoiceResponseObject, error) {
	if req.Body == nil {
		return inv422Create("Request body is required.", nil), nil
	}

	total, paid, errs := parseInvoiceAmounts(req.Body.TotalAmount, req.Body.AmountPaid)
	for k, v := range validateInvoiceCreate(*req.Body) {
		errs[k] = v
	}
	if len(errs) == 0 {
		// Cross-field check only meaningful when both parsed cleanly.
		if paid.GreaterThan(total) {
			errs["amount_paid"] = []string{"Amount paid cannot exceed total amount."}
		}
	}
	if len(errs) > 0 {
		return inv422Create("The given data was invalid.", errs), nil
	}

	if fkErrs := s.validateInvoiceFKAndUnique(ctx, req.Body.AppointmentId, 0); len(fkErrs) > 0 {
		return inv422Create("The given data was invalid.", fkErrs), nil
	}

	m := fromCreateInvoice(*req.Body, total, paid)
	if err := s.DB.WithContext(ctx).Create(&m).Error; err != nil {
		if errs := translateInvoiceMySQLError(err); errs != nil {
			return inv422Create("The given data was invalid.", errs), nil
		}
		return nil, err
	}
	// Re-fetch so AfterFind populates RemainingBalance and Appointment/Patient eager-load.
	if err := s.DB.WithContext(ctx).Preload("Appointment.Patient").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return CreateInvoice201JSONResponse{Data: toAPIInvoice(&m)}, nil
}

func (s *Server) GetInvoice(ctx context.Context, req GetInvoiceRequestObject) (GetInvoiceResponseObject, error) {
	var m models.Invoice
	if err := s.DB.WithContext(ctx).Preload("Appointment.Patient").First(&m, req.Id).Error; err != nil {
		return nil, err
	}
	return GetInvoice200JSONResponse{Data: toAPIInvoice(&m)}, nil
}

func (s *Server) UpdateInvoice(ctx context.Context, req UpdateInvoiceRequestObject) (UpdateInvoiceResponseObject, error) {
	if req.Body == nil {
		return inv422Update("Request body is required.", nil), nil
	}

	var m models.Invoice
	if err := s.DB.WithContext(ctx).First(&m, req.Id).Error; err != nil {
		return nil, err
	}

	// Resolve effective post-merge amounts so the invariant (paid ≤ total)
	// can be enforced even if the caller only sent one of the two fields.
	effTotal := m.TotalAmount
	effPaid := m.AmountPaid
	errs := validateInvoiceUpdate(*req.Body)
	if req.Body.TotalAmount != nil {
		v, err := decimal.NewFromString(*req.Body.TotalAmount)
		switch {
		case err != nil:
			errs["total_amount"] = []string{"Must be a decimal number."}
		case v.IsNegative():
			errs["total_amount"] = []string{"Must be ≥ 0."}
		default:
			effTotal = v
		}
	}
	if req.Body.AmountPaid != nil {
		v, err := decimal.NewFromString(*req.Body.AmountPaid)
		switch {
		case err != nil:
			errs["amount_paid"] = []string{"Must be a decimal number."}
		case v.IsNegative():
			errs["amount_paid"] = []string{"Must be ≥ 0."}
		default:
			effPaid = v
		}
	}
	if len(errs) == 0 && effPaid.GreaterThan(effTotal) {
		errs["amount_paid"] = []string{"Amount paid cannot exceed total amount."}
	}
	if len(errs) > 0 {
		return inv422Update("The given data was invalid.", errs), nil
	}

	applyInvoiceUpdate(&m, *req.Body, effTotal, effPaid)
	if err := s.DB.WithContext(ctx).Save(&m).Error; err != nil {
		if errs := translateInvoiceMySQLError(err); errs != nil {
			return inv422Update("The given data was invalid.", errs), nil
		}
		return nil, err
	}
	if err := s.DB.WithContext(ctx).Preload("Appointment.Patient").First(&m, m.ID).Error; err != nil {
		return nil, err
	}
	return UpdateInvoice200JSONResponse{Data: toAPIInvoice(&m)}, nil
}

func (s *Server) DeleteInvoice(ctx context.Context, req DeleteInvoiceRequestObject) (DeleteInvoiceResponseObject, error) {
	res := s.DB.WithContext(ctx).Delete(&models.Invoice{}, req.Id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return DeleteInvoice204Response{}, nil
}

// --- validation -------------------------------------------------------------

// parseInvoiceAmounts parses total_amount + amount_paid (both decimal-as-
// string) into Decimal values, returning per-field errors for anything
// non-numeric or negative.
func parseInvoiceAmounts(totalStr string, paidPtr *string) (total, paid decimal.Decimal, errs map[string][]string) {
	errs = map[string][]string{}
	if totalStr == "" {
		errs["total_amount"] = []string{"The total_amount field is required."}
	} else {
		v, err := decimal.NewFromString(totalStr)
		switch {
		case err != nil:
			errs["total_amount"] = []string{"Must be a decimal number."}
		case v.IsNegative():
			errs["total_amount"] = []string{"Must be ≥ 0."}
		default:
			total = v
		}
	}
	if paidPtr != nil && *paidPtr != "" {
		v, err := decimal.NewFromString(*paidPtr)
		switch {
		case err != nil:
			errs["amount_paid"] = []string{"Must be a decimal number."}
		case v.IsNegative():
			errs["amount_paid"] = []string{"Must be ≥ 0."}
		default:
			paid = v
		}
	}
	return
}

func validateInvoiceCreate(b InvoiceCreate) map[string][]string {
	errs := map[string][]string{}
	if b.AppointmentId <= 0 {
		errs["appointment_id"] = []string{"The appointment_id field is required."}
	}
	if b.InvoiceDate.Time.IsZero() {
		errs["invoice_date"] = []string{"The invoice_date field is required."}
	}
	if b.PaymentStatus != nil && !b.PaymentStatus.Valid() {
		errs["payment_status"] = []string{"Selected payment_status is invalid."}
	}
	return errs
}

func validateInvoiceUpdate(b InvoiceUpdate) map[string][]string {
	errs := map[string][]string{}
	if b.PaymentStatus != nil && !b.PaymentStatus.Valid() {
		errs["payment_status"] = []string{"Selected payment_status is invalid."}
	}
	return errs
}

// validateInvoiceFKAndUnique checks (a) the appointment exists and
// (b) no other invoice already points at it. excludeInvoiceID is the
// current invoice's id during an update (so an in-place edit doesn't
// trigger its own uniqueness rule); pass 0 on create.
func (s *Server) validateInvoiceFKAndUnique(ctx context.Context, appointmentID int64, excludeInvoiceID uint) map[string][]string {
	errs := map[string][]string{}
	var c int64
	if err := s.DB.WithContext(ctx).Model(&models.Appointment{}).Where("id = ?", appointmentID).Count(&c).Error; err == nil && c == 0 {
		errs["appointment_id"] = []string{"Appointment does not exist."}
		return errs // no point checking uniqueness against a non-existent FK
	}
	q := s.DB.WithContext(ctx).Model(&models.Invoice{}).Where("appointment_id = ?", appointmentID)
	if excludeInvoiceID > 0 {
		q = q.Where("id <> ?", excludeInvoiceID)
	}
	if err := q.Count(&c).Error; err == nil && c > 0 {
		errs["appointment_id"] = []string{"An invoice already exists for this appointment."}
	}
	return errs
}

// translateInvoiceMySQLError converts the two MySQL codes invoices can
// trigger (1062 on the appointment_id unique index, 1452 on the FK) into
// Laravel-shaped field errors. Returns nil otherwise so the caller
// bubbles up to a 500.
func translateInvoiceMySQLError(err error) map[string][]string {
	var mErr *gomysql.MySQLError
	if !errors.As(err, &mErr) {
		return nil
	}
	switch mErr.Number {
	case 1062:
		return map[string][]string{
			"appointment_id": {"An invoice already exists for this appointment."},
		}
	case 1452:
		return map[string][]string{
			"appointment_id": {"Appointment does not exist."},
		}
	}
	return nil
}

func inv422Create(message string, fieldErrs map[string][]string) CreateInvoice422JSONResponse {
	if fieldErrs == nil {
		fieldErrs = map[string][]string{}
	}
	return CreateInvoice422JSONResponse{ValidationErrorJSONResponse{
		Message: message,
		Errors:  fieldErrs,
	}}
}

func inv422Update(message string, fieldErrs map[string][]string) UpdateInvoice422JSONResponse {
	if fieldErrs == nil {
		fieldErrs = map[string][]string{}
	}
	return UpdateInvoice422JSONResponse{ValidationErrorJSONResponse{
		Message: message,
		Errors:  fieldErrs,
	}}
}

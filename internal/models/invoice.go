package models

import (
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "Pending"
	PaymentStatusPaid    PaymentStatus = "Paid"
	PaymentStatusPartial PaymentStatus = "Partial"
)

type Invoice struct {
	Base
	AppointmentID uint            `gorm:"not null;uniqueIndex:invoices_appointment_id_unique" json:"appointment_id"`
	InvoiceDate   datatypes.Date  `gorm:"type:date;not null" json:"invoice_date"`
	TotalAmount   decimal.Decimal `gorm:"type:decimal(10,2);not null" json:"total_amount"`
	AmountPaid    decimal.Decimal `gorm:"type:decimal(10,2);not null;default:0.00" json:"amount_paid"`
	PaymentStatus PaymentStatus   `gorm:"type:enum('Pending','Paid','Partial');not null;default:'Pending'" json:"payment_status"`

	// RemainingBalance = TotalAmount - AmountPaid; matches the Backpack
	// admin grid's `remaining_balance` closure column.
	RemainingBalance decimal.Decimal `gorm:"-" json:"remaining_balance"`

	Appointment *Appointment `gorm:"foreignKey:AppointmentID" json:"appointment,omitempty"`
}

func (i *Invoice) AfterFind(_ *gorm.DB) error {
	i.RemainingBalance = i.TotalAmount.Sub(i.AmountPaid)
	return nil
}

func (i Invoice) IsPaid() bool { return i.PaymentStatus == PaymentStatusPaid }

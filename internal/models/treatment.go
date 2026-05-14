package models

import "github.com/shopspring/decimal"

type Treatment struct {
	Base
	CategoryID   *uint           `json:"category_id,omitempty"`
	Name         string          `gorm:"size:100;not null;uniqueIndex:treatments_name_unique" json:"name"`
	Description  *string         `gorm:"type:text" json:"description,omitempty"`
	StandardCost decimal.Decimal `gorm:"type:decimal(10,2);not null" json:"standard_cost"`

	Category     *Category     `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	Appointments []Appointment `gorm:"many2many:visit_treatments;" json:"appointments,omitempty"`
}

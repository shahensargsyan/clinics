// Package models contains GORM struct definitions that mirror the existing
// Laravel/Backpack MySQL schema 1:1. JSON tags use snake_case to preserve
// API-response parity with the Laravel controllers we're migrating from.
package models

import (
	"time"

	"gorm.io/gorm"
)

// Base mirrors gorm.Model but ships explicit json tags so output matches
// Laravel/Backpack's snake_case conventions. Embed in any model whose table
// carries the canonical id + created_at + updated_at + deleted_at columns.
type Base struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// Timestamped is for tables that have id + created_at + updated_at but no
// deleted_at column (cephalometric_analyses, diagnostic_assets,
// mechanotherapy_visits, tooth_measurements, permissions, roles).
type Timestamped struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

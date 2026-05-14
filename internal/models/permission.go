package models

// Role, Permission, and the three pivot tables map Spatie's
// laravel-permission package 1:1 so the existing DB can be shared during
// the Laravel → Go migration window. Step 4 will plug these into a Casbin
// (or hand-rolled) RBAC middleware.

type Role struct {
	Timestamped
	Name      string `gorm:"size:255;not null;uniqueIndex:roles_name_guard_name_unique,priority:1" json:"name"`
	GuardName string `gorm:"size:255;not null;uniqueIndex:roles_name_guard_name_unique,priority:2" json:"guard_name"`

	Permissions []Permission `gorm:"many2many:role_has_permissions;" json:"permissions,omitempty"`
}

type Permission struct {
	Timestamped
	Name      string `gorm:"size:255;not null;uniqueIndex:permissions_name_guard_name_unique,priority:1" json:"name"`
	GuardName string `gorm:"size:255;not null;uniqueIndex:permissions_name_guard_name_unique,priority:2" json:"guard_name"`

	Roles []Role `gorm:"many2many:role_has_permissions;" json:"roles,omitempty"`
}

// RoleHasPermission is the role↔permission join. GORM manages it via the
// many2many tags above; we expose the struct so direct queries are
// possible when implementing authorization checks.
type RoleHasPermission struct {
	PermissionID uint `gorm:"primaryKey" json:"permission_id"`
	RoleID       uint `gorm:"primaryKey" json:"role_id"`
}

// ModelHasPermission is Spatie's polymorphic permission grant. model_type
// stores a fully-qualified Laravel class string (e.g. "App\\Models\\User").
// When the Go service becomes the sole writer, standardize this on a
// stable constant rather than a Laravel class name.
type ModelHasPermission struct {
	PermissionID uint   `gorm:"primaryKey" json:"permission_id"`
	ModelType    string `gorm:"primaryKey;size:255" json:"model_type"`
	ModelID      uint   `gorm:"primaryKey" json:"model_id"`
}

type ModelHasRole struct {
	RoleID    uint   `gorm:"primaryKey" json:"role_id"`
	ModelType string `gorm:"primaryKey;size:255" json:"model_type"`
	ModelID   uint   `gorm:"primaryKey" json:"model_id"`
}

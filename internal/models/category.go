package models

// Category is the treatment-categorization tree used by Backpack. The schema
// keeps parent_id nullable with a 0 default, matching Laravel's convention
// of "0 means root"; we preserve that here rather than introducing a
// breaking sentinel change.
type Category struct {
	Base
	ParentID uint   `gorm:"default:0" json:"parent_id"`
	Name     string `gorm:"size:255;not null" json:"name"`
	Slug     string `gorm:"size:255;not null;uniqueIndex:categories_slug_unique" json:"slug"`

	Parent     *Category   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children   []Category  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Treatments []Treatment `gorm:"foreignKey:CategoryID" json:"treatments,omitempty"`
}

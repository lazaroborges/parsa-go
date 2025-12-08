package tag

import (
	"errors"
	"time"
)

var (
	ErrTagNotFound = errors.New("tag not found")
	ErrForbidden   = errors.New("forbidden: tag does not belong to user")
)

type Tag struct {
	ID           string    `json:"id"`
	UserID       int64     `json:"-"`
	Name         string    `json:"name"`
	Color        string    `json:"color"`
	DisplayOrder int       `json:"displayOrder"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateTagParams struct {
	Name         string
	Color        string
	DisplayOrder *int
	Description  *string
}

func (p *CreateTagParams) Validate() error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	if len(p.Name) > 128 {
		return errors.New("name must be 128 characters or less")
	}
	if p.Color == "" {
		return errors.New("color is required")
	}
	if len(p.Color) > 12 {
		return errors.New("color must be 12 characters or less")
	}
	if p.Description != nil && len(*p.Description) > 255 {
		return errors.New("description must be 255 characters or less")
	}
	return nil
}

type UpdateTagParams struct {
	Name         *string
	Color        *string
	DisplayOrder *int
	Description  *string
}

func (p *UpdateTagParams) Validate() error {
	if p.Name != nil && len(*p.Name) > 128 {
		return errors.New("name must be 128 characters or less")
	}
	if p.Color != nil && len(*p.Color) > 12 {
		return errors.New("color must be 12 characters or less")
	}
	if p.Description != nil && len(*p.Description) > 255 {
		return errors.New("description must be 255 characters or less")
	}
	return nil
}

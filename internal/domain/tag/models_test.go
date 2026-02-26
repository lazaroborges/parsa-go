package tag

import (
	"strings"
	"testing"
)

func TestCreateTagParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  CreateTagParams
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid params",
			params:  CreateTagParams{Name: "Work", Color: "#FF0000"},
			wantErr: false,
		},
		{
			name:    "missing name",
			params:  CreateTagParams{Name: "", Color: "#FF0000"},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "name too long",
			params:  CreateTagParams{Name: strings.Repeat("a", 129), Color: "#FF0000"},
			wantErr: true,
			errMsg:  "name must be 128 characters or less",
		},
		{
			name:    "name exactly 128 chars",
			params:  CreateTagParams{Name: strings.Repeat("a", 128), Color: "#FF0000"},
			wantErr: false,
		},
		{
			name:    "missing color",
			params:  CreateTagParams{Name: "Work", Color: ""},
			wantErr: true,
			errMsg:  "color is required",
		},
		{
			name:    "color too long",
			params:  CreateTagParams{Name: "Work", Color: strings.Repeat("a", 13)},
			wantErr: true,
			errMsg:  "color must be 12 characters or less",
		},
		{
			name: "description too long",
			params: CreateTagParams{
				Name:        "Work",
				Color:       "#FF0000",
				Description: strPtr(strings.Repeat("a", 256)),
			},
			wantErr: true,
			errMsg:  "description must be 255 characters or less",
		},
		{
			name: "valid with optional fields",
			params: CreateTagParams{
				Name:         "Work",
				Color:        "#FF0000",
				DisplayOrder: intPtr(1),
				Description:  strPtr("My work tag"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUpdateTagParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  UpdateTagParams
		wantErr bool
		errMsg  string
	}{
		{
			name:    "all nil (no updates)",
			params:  UpdateTagParams{},
			wantErr: false,
		},
		{
			name:    "valid name update",
			params:  UpdateTagParams{Name: strPtr("New Name")},
			wantErr: false,
		},
		{
			name:    "empty name",
			params:  UpdateTagParams{Name: strPtr("")},
			wantErr: true,
			errMsg:  "name cannot be empty",
		},
		{
			name:    "name too long",
			params:  UpdateTagParams{Name: strPtr(strings.Repeat("a", 129))},
			wantErr: true,
			errMsg:  "name must be 128 characters or less",
		},
		{
			name:    "empty color",
			params:  UpdateTagParams{Color: strPtr("")},
			wantErr: true,
			errMsg:  "color cannot be empty",
		},
		{
			name:    "color too long",
			params:  UpdateTagParams{Color: strPtr(strings.Repeat("a", 13))},
			wantErr: true,
			errMsg:  "color must be 12 characters or less",
		},
		{
			name:    "negative display order",
			params:  UpdateTagParams{DisplayOrder: intPtr(-1)},
			wantErr: true,
			errMsg:  "display order must be non-negative",
		},
		{
			name:    "zero display order",
			params:  UpdateTagParams{DisplayOrder: intPtr(0)},
			wantErr: false,
		},
		{
			name:    "description too long",
			params:  UpdateTagParams{Description: strPtr(strings.Repeat("a", 256))},
			wantErr: true,
			errMsg:  "description must be 255 characters or less",
		},
		{
			name: "valid full update",
			params: UpdateTagParams{
				Name:         strPtr("Updated"),
				Color:        strPtr("#00FF00"),
				DisplayOrder: intPtr(5),
				Description:  strPtr("Updated description"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Validate() error = %q, want %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

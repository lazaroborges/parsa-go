package account

import (
	"testing"
)

func TestIsValidAccountType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"BANK", true},
		{"CREDIT", true},
		{"INVESTMENT", true},
		{"INVALID", false},
		{"bank", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsValidAccountType(tt.input)
			if got != tt.want {
				t.Errorf("IsValidAccountType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidAccountSubtype(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"CHECKING_ACCOUNT", true},
		{"SAVINGS_ACCOUNT", true},
		{"CREDIT_CARD", true},
		{"INVALID", false},
		{"checking_account", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsValidAccountSubtype(tt.input)
			if got != tt.want {
				t.Errorf("IsValidAccountSubtype(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsValidCurrency(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"BRL", true},
		{"USD", true},
		{"EUR", true},
		{"GBP", true},
		{"JPY", true},
		{"INVALID", false},
		{"usd", false},
		{"US", false},
		{"", false},
		{"ABCD", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := IsValidCurrency(tt.input)
			if got != tt.want {
				t.Errorf("IsValidCurrency(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  CreateParams
		wantErr bool
		errType error
	}{
		{
			name: "valid params",
			params: CreateParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			params: CreateParams{
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
			},
			wantErr: true,
		},
		{
			name: "invalid user ID",
			params: CreateParams{
				ID:          "acc-1",
				UserID:      0,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			params: CreateParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "",
				AccountType: "BANK",
				Currency:    "USD",
			},
			wantErr: true,
		},
		{
			name: "missing account type",
			params: CreateParams{
				ID:       "acc-1",
				UserID:   1,
				Name:     "Test",
				Currency: "USD",
			},
			wantErr: true,
		},
		{
			name: "invalid account type",
			params: CreateParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "UNKNOWN",
				Currency:    "USD",
			},
			wantErr: true,
			errType: ErrInvalidAccountType,
		},
		{
			name: "missing currency",
			params: CreateParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "",
			},
			wantErr: true,
		},
		{
			name: "invalid currency",
			params: CreateParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "XYZ",
			},
			wantErr: true,
			errType: ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("Validate() error = %v, want %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUpsertParams_Validate(t *testing.T) {
	validSubtype := "CHECKING_ACCOUNT"
	invalidSubtype := "INVALID"

	tests := []struct {
		name    string
		params  UpsertParams
		wantErr bool
		errType error
	}{
		{
			name: "valid params",
			params: UpsertParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
			},
			wantErr: false,
		},
		{
			name: "valid with subtype",
			params: UpsertParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
				Subtype:     &validSubtype,
			},
			wantErr: false,
		},
		{
			name: "invalid subtype",
			params: UpsertParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
				Subtype:     &invalidSubtype,
			},
			wantErr: true,
			errType: ErrInvalidAccountSubtype,
		},
		{
			name: "missing ID",
			params: UpsertParams{
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
			},
			wantErr: true,
		},
		{
			name: "invalid user ID",
			params: UpsertParams{
				ID:          "acc-1",
				UserID:      0,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "USD",
			},
			wantErr: true,
		},
		{
			name: "invalid currency",
			params: UpsertParams{
				ID:          "acc-1",
				UserID:      1,
				Name:        "Test",
				AccountType: "BANK",
				Currency:    "INVALID",
			},
			wantErr: true,
			errType: ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("Validate() expected error, got nil")
				}
				if tt.errType != nil && err != tt.errType {
					t.Errorf("Validate() error = %v, want %v", err, tt.errType)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

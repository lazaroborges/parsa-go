package transaction

import (
	"testing"
)

func TestGetCategoryKey_WithCode(t *testing.T) {
	code := "01000000"
	result := GetCategoryKey(&code)
	if result == nil {
		t.Fatal("GetCategoryKey() returned nil for valid code")
	}
	if *result != "01000000" {
		t.Errorf("GetCategoryKey() = %q, want %q", *result, "01000000")
	}
}

func TestGetCategoryKey_WithOpenFinanceName(t *testing.T) {
	name := "Renda"
	result := GetCategoryKey(&name)
	if result == nil {
		t.Fatal("GetCategoryKey() returned nil for valid OpenFinanceName")
	}
	if *result != "01000000" {
		t.Errorf("GetCategoryKey() = %q, want %q", *result, "01000000")
	}
}

func TestGetCategoryKey_Nil(t *testing.T) {
	result := GetCategoryKey(nil)
	if result != nil {
		t.Errorf("GetCategoryKey(nil) = %q, want nil", *result)
	}
}

func TestGetCategoryKey_EmptyString(t *testing.T) {
	empty := ""
	result := GetCategoryKey(&empty)
	if result != nil {
		t.Errorf("GetCategoryKey(\"\") = %q, want nil", *result)
	}
}

func TestGetCategoryKey_UnknownCategory(t *testing.T) {
	unknown := "NonExistentCategory"
	result := GetCategoryKey(&unknown)
	if result != nil {
		t.Errorf("GetCategoryKey(%q) = %q, want nil", unknown, *result)
	}
}

func TestTranslateCategory_WithCode(t *testing.T) {
	code := "01000000"
	result := TranslateCategory(&code)
	if result == nil {
		t.Fatal("TranslateCategory() returned nil for valid code")
	}
	if *result != "Renda Ativa" {
		t.Errorf("TranslateCategory() = %q, want %q", *result, "Renda Ativa")
	}
}

func TestTranslateCategory_WithOpenFinanceName(t *testing.T) {
	name := "Salário"
	result := TranslateCategory(&name)
	if result == nil {
		t.Fatal("TranslateCategory() returned nil for valid OpenFinanceName")
	}
	if *result != "Salário" {
		t.Errorf("TranslateCategory() = %q, want %q", *result, "Salário")
	}
}

func TestTranslateCategory_Nil(t *testing.T) {
	result := TranslateCategory(nil)
	if result != nil {
		t.Errorf("TranslateCategory(nil) should return nil")
	}
}

func TestTranslateCategory_EmptyString(t *testing.T) {
	empty := ""
	result := TranslateCategory(&empty)
	if result != nil {
		t.Errorf("TranslateCategory(\"\") should return nil")
	}
}

func TestTranslateCategory_UnknownFallback(t *testing.T) {
	unknown := "Custom Category"
	result := TranslateCategory(&unknown)
	if result == nil {
		t.Fatal("TranslateCategory() returned nil for unknown category")
	}
	if *result != unknown {
		t.Errorf("TranslateCategory(%q) = %q, want fallback %q", unknown, *result, unknown)
	}
}

func TestTranslateCategory_SpecificMappings(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"10000000", "Mercado"},
		{"11020000", "Delivery"},
		{"12050000", "Combustível"},
		{"05100000", "Pagamento Fatura do Cartão"},
		{"19010000", "Táxi e transporte privado urbano"},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := TranslateCategory(&tt.code)
			if result == nil {
				t.Fatalf("TranslateCategory(%q) returned nil", tt.code)
			}
			if *result != tt.expected {
				t.Errorf("TranslateCategory(%q) = %q, want %q", tt.code, *result, tt.expected)
			}
		})
	}
}

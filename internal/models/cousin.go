// Cousin represents a unique counterpart (merchant or individual - CNPJ or CPF) in the real world and a set of things that help identify them (Document, merchant object, description patterns, etc.). This doesn't have to with persons and family relations.
//
// Fields:
//   - ID: Unique identifier (BIGINT).
//   - Document: Document object (CPF or CNPJ).
//   - BusinessName: Full business name as per registered in the Receita Federal records.
//   - Name: The business public name ("Nome de Fantasia").

package models

type Cousin struct {
	ID           int64    `json:"id"`
	Document     Document `json:"document"`
	BusinessName string   `json:"businessName"`
	Name         string   `json:"name"`
}

package models

const (
	DocumentTypeCPF  = "cpf"
	DocumentTypeCNPJ = "cnpj"
)

type Document struct {
	ID           int64   `json:"id"`
	Type         string  `json:"type"`                   // "cpf" or "cnpj"
	Number       *string `json:"number,omitempty"`        // Brazilian document number as string (nullable)
	BusinessName *string `json:"businessName,omitempty"`
}

type CreateDocumentParams struct {
	Type   string
	Number string
}

// ValidateType checks if the document type is valid
func (d *Document) ValidateType() bool {
	return d.Type == DocumentTypeCPF || d.Type == DocumentTypeCNPJ
}

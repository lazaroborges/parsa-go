package models

const (
	DocumentTypeCPF  = "cpf"
	DocumentTypeCNPJ = "cnpj"
)

type Document struct {
	Type   string `json:"type"`   // "cpf" or "cnpj"
	Number string `json:"number"` // Brazilian document number as string
}

type CreateDocumentParams struct {
	Type   string
	Number string
}

// ValidateType checks if the document type is valid
func (d *Document) ValidateType() bool {
	return d.Type == DocumentTypeCPF || d.Type == DocumentTypeCNPJ
}

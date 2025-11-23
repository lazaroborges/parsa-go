package models

type Bank struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connector string `json:"connectorId"`
}

type CreateBankParams struct {
	Name      string
	Connector string
}

type UpdateBankParams struct {
	Name      *string
	Connector *string
}

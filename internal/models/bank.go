package models

type Bank struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Connector string `json:"connector_id"`
}

type CreateBankParams struct {
	Name      string
	Connector string
}

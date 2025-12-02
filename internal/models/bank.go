package models

type Bank struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	UIName       string `json:"uiName"`
	Connector    string `json:"connectorId"`
	PrimaryColor string `json:"primaryColor"`
}

type CreateBankParams struct {
	Name      string
	Connector string
}

type UpdateBankParams struct {
	Name      *string
	Connector *string
}

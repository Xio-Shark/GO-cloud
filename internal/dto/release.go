package dto

type CreateReleaseRequest struct {
	AppName     string `json:"app_name"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	Operator    string `json:"operator"`
	ChangeLog   string `json:"change_log"`
}

type RollbackReleaseRequest struct {
	Operator  string `json:"operator"`
	ChangeLog string `json:"change_log"`
}

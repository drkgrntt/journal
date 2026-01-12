package models

func init() {
	registerModel(&Analytic{})
}

type Analytic struct {
	*Base
	Domain    string `json:"domain,omitempty"`
	Page      string `json:"page,omitempty"`
	Query     string `json:"query,omitempty"`
	IP        string `json:"ip,omitempty"`
	Useragent string `json:"useragent,omitempty"`
}

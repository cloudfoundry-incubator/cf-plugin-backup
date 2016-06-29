package models

type ResourceCollectionModel struct {
	TotalResults int               `json:"total_results"`
	TotalPages   int               `json:"total_pages"`
	NextURL      string            `json:"next_url,omitempty"`
	PrevURL      string            `json:"prev_url,omitempty"`
	Resources    *[]*ResourceModel `json:"resources"`
}

type ResourceModel struct {
	Metadata map[string]interface{} `json:"metadata"`
	Entity   map[string]interface{} `json:"entity"`
}

type BackupModel struct {
	Organizations  interface{} `json:"organizations"`
	SharedDomains  interface{} `json:"shared_domains"`
	SecurityGroups interface{} `json:"security_groups"`
}

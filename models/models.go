package models

// ResourceCollectionModel represents the paged response of an api call
type ResourceCollectionModel struct {
	TotalResults int               `json:"total_results"`
	TotalPages   int               `json:"total_pages"`
	NextURL      string            `json:"next_url,omitempty"`
	PrevURL      string            `json:"prev_url,omitempty"`
	Resources    *[]*ResourceModel `json:"resources"`
}

// ResourceModel represents the response of an api call
type ResourceModel struct {
	Metadata map[string]interface{} `json:"metadata"`
	Entity   map[string]interface{} `json:"entity"`
}

// BackupModel represents the backup json model
type BackupModel struct {
	OrgQuotas      interface{} `json:"org_quota_definitions"`
	Organizations  interface{} `json:"organizations"`
	SharedDomains  interface{} `json:"shared_domains"`
	SecurityGroups interface{} `json:"security_groups"`
	FeatureFlags   interface{} `json:"feature_flags"`
}

// FeatureFlagModel represents the feature flag json model
type FeatureFlagModel struct {
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	ErrorMessage string `json:"error_message,omitempty"`
	URL          string `json:"url"`
}

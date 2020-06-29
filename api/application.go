package api

type Application struct {
	Id               string `json:"id,omitempty"`
	Name             string `json:"name"`
	CreatedAt        int    `json:"created_at,omitempty"`
	LastDeploymentAt int    `json:"last_deployment_at,omitempty"`
}

type ApplicationDetails struct {
	Id          string                   `json:"id,omitempty"`
	CreatedAt   int                      `json:"created_at,omitempty"`
	Status      string                   `json:"status"`
	ErrorReason string                   `json:"error_reason"`
	Services    map[string]ServiceStatus `json:"services"`
}

type ServiceStatus struct {
	Image     string   `json:"image"`
	Endpoints []string `json:"endpoints"`
	Status    string   `json:"status"`
	Errors    []string `json:"errors"`
}

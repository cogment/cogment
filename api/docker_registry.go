package api

type DockerRegistry struct {
	Id          int    `json:"id,omitempty"`
	RegistryUrl string `json:"registry_url"`
	CreatedAt   int    `json:"created_at,omitempty"`
	UpdatedAt   int    `json:"updated_at,omitempty"`
}
type RegistryConfiguration struct {
	RegistryUrl string `json:"registry_url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

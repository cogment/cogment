package templates

const ROOT_DATA_PROTO = `
syntax = "proto3";

package {{.ProjectName}};

message EnvConfig {
}

message TrialConfig {
	EnvConfig env_config = 1;
}

message Observation {}

{{- range .ActorClasses}}
message {{.Id|pascalify}}Action {}
{{end}}
`

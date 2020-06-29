package templates

const ROOT_DATA_PROTO = `
syntax = "proto3";

package bootstrap;

message Observation {}

{{- range .ActorClasses}}
message {{.Id|pascalify}}Action {}
{{end}}
`

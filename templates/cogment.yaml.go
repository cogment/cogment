package templates

const ROOT_COGMENT_YAML = `
import:
  proto:
    - data.proto

commands:
  build: cogment -v generate --python_dir=. && docker-compose build
  start: docker-compose up orchestrator env {{- range .ActorClasses}} {{.Id}} {{end}}
  stop: docker-compose stop orchestrator env {{- range .ActorClasses}} {{.Id}} {{end}}
  client: docker-compose run --rm client


# Static configuration
actor_classes:
{{- range .ActorClasses}}
  - id: {{.Id|snakeify}}
    action:
      space: bootstrap.{{.Id|pascalify}}Action
    observation:
      space: bootstrap.Observation
{{end}}

# Dynamic configuration (could be changed by a pre-hook)
trial_params:
  environment:
    endpoint: grpc://env:9000

  actors:
  {{- range .TrialParams.Actors}}
    - actor_class: {{.ActorClass|snakeify}}
      endpoint: {{.Endpoint}}
  {{ end }}
`

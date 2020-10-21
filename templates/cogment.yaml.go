// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templates

const ROOT_COGMENT_YAML = `{{ $config := . }}
import:
  proto:
    - data.proto

commands:
  proto: cogment -v generate --python_dir=. --js_dir=clients/js
  build: cogment -v generate --python_dir=. && docker-compose build
  # python client
  start: docker-compose up orchestrator env  {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}}
  stop: docker-compose stop orchestrator env  {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}}
  # web client
  start-webui: docker-compose up orchestrator webui env  {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}} envoy
  stop-webui: docker-compose stop orchestrator webui env  {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}} envoy
  # python client with configurator
  #start-configurator: docker-compose up orchestrator env  {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}}
  #stop-configurator: docker-compose stop orchestrator env  {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}}
  # web client with configurator
  #start-webui-configurator: docker-compose up orchestrator webui env configurator {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}} envoy
  #stop-webui-configurator: docker-compose stop orchestrator webui env configurator {{- range .ActorClasses}} {{- if $config.HasAiByActorClass .Id }} {{.Id}} {{end}}{{end}} envoy
  client: docker-compose run --rm client
  # Log exporter with postgres
  #create-db-volume: docker volume create postgres_database
  #export-data: docker-compose up log_exporter

{{$projectname := .ProjectName}}

environment:
  config_type: {{$projectname}}.EnvConfig

trial:
  config_tyep: {{$projectname}}.TrialConfig
#  pre_hooks:
#    - grpc://configurator:9000


# Static configuration
actor_classes:
{{- range .ActorClasses}}
  - id: {{.Id|snakeify}}
    action:
      space: {{$projectname}}.{{.Id|pascalify}}Action
    observation:
      space: {{$projectname}}.Observation
{{end}}

# Dynamic configuration (could be changed by a pre-hook)
trial_params:
  #max_inactivity: 2
  environment:
    endpoint: grpc://env:9000
    #config: {}
    
  actors:
  {{- range .TrialParams.Actors}}
    - actor_class: {{.ActorClass|snakeify}}
      endpoint: {{.Endpoint}}
  {{ end }}

# Log exporter & postgres database
#datalog:
#  type: grpc
#  url: log_exporter:9000

# Log export & redis
#datalog:
#  type: grpc
#  url: replaybuffer:9000
`

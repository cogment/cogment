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

const ROOT_DOCKER_COMPOSE_OVERRIDE = `
{{ $config := . }}

version: '3.7'

services:
  
{{- range .ActorClasses}}

{{- if $config.HasAiByActorClass .Id }}
  {{.Id|kebabify}}:
    volumes:
      - ./:/app
{{end}}
{{end}}

  env:
    volumes:
      - ./:/app
    environment: 
      - COGMENT_GRPC_REFLECTION=1
      - PYTHONUNBUFFERED=1

  orchestrator:
    volumes:
      - .:/app
    ports:
      - "9000:9000"

  client:
    volumes:
      - ./:/app

# log exporter with postgres db - need to add lan IP address
# 
#  setup_db:
#      environment:
#        - IP= # Please specify your computer's LAN!
#  truncate_db:
#      environment:
#        - IP= # Please specify your computer's LAN!

`

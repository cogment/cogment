package templates

const ROOT_DOCKER_COMPOSE = `
{{ $config := . }}

version: '3.7'

services:
  grpc-cli:
    image: namely/grpc-cli 
  
  cogment-cli:
    image: registry.gitlab.com/cogment/cogment/cli
    volumes:
      - ./:/app


{{- range .ActorClasses}}

{{- if $config.HasAiByActorClass .Id }}
  {{.Id|kebabify}}:
    image: registry.gitlab.com/change_me/{{.Id|snakeify}}
    build: 
      context: .
      dockerfile: agents/{{.Id|snakeify}}/Dockerfile
    volumes:
      - ./:/app
    environment: 
      - COGMENT_GRPC_REFLECTION=1
      - PYTHONUNBUFFERED=1
    x-cogment:
      ports:
        - port: 9000
{{end}}
{{end}}


  env:
    image: registry.gitlab.com/change_me/env
    build: 
      context: .
      dockerfile: envs/Dockerfile
    volumes:
      - ./:/app
    environment: 
      - COGMENT_GRPC_REFLECTION=1
      - PYTHONUNBUFFERED=1
    x-cogment:
      ports:
        - port: 9000

  orchestrator:
    image: registry.gitlab.com/change_me/orchestrator
    build: 
      context: .
      dockerfile: orchestrator/Dockerfile
    volumes:
      - .:/app
    ports:
      - "9000:9000"
    x-cogment:
      ports:
        - port: 9000
    depends_on:
      - env
      {{- range .ActorClasses}}
      {{- if $config.HasAiByActorClass .Id }}
      - {{.Id|kebabify}}
      {{- end }}
      {{end}}

  client:
    build: 
      context: .
      dockerfile: clients/Dockerfile
    volumes:
      - ./:/app
    environment: 
      - PYTHONUNBUFFERED=1
    stdin_open: true
    tty: true          
`

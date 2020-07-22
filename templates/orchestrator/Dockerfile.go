package orchestrator

const DOCKERFILE = `
FROM registry.gitlab.com/cogment/cogment/orchestrator:0.3.0-alpha6

ADD cogment.yaml .
ADD *.proto .
`

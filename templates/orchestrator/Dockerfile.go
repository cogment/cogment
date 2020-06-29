package orchestrator

const DOCKERFILE = `
FROM registry.gitlab.com/cogment/cogment/orchestrator:0.2

ADD cogment.yaml .
ADD *.proto .
`

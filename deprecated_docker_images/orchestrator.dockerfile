ARG COGMENT_IMAGE=cogment/cogment:latest
FROM ${COGMENT_IMAGE}

ENTRYPOINT ["cogment", "services", "orchestrator"]

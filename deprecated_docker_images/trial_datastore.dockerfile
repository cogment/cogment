ARG COGMENT_IMAGE=cogment/cogment:latest
FROM ${COGMENT_IMAGE}

ENV COGMENT_TRIAL_DATASTORE_PORT=9000

ENTRYPOINT ["cogment", "services", "trial_datastore"]

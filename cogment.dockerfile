FROM ubuntu:20.04

ARG COGMENT_EXEC=./install/linux_amd64/bin/cogment
COPY ${COGMENT_EXEC} /usr/local/bin/cogment
RUN chmod +x /usr/local/bin/cogment

VOLUME /data
ENV COGMENT_MODEL_REGISTRY_ARCHIVE_DIR=/data/model_registry

ENTRYPOINT ["cogment"]

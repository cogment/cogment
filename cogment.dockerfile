ARG COGMENT_EXEC=./install/linux_amd64/bin/cogment

FROM ubuntu:20.04

COPY ${COGMENT_EXEC} /usr/local/bin/cogment

VOLUME /data
ENV COGMENT_MODEL_REGISTRY_ARCHIVE_DIR=/data

ENTRYPOINT ["cogment"]

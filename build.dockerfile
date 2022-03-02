ARG COGMENT_BUILD_ENVIRONMENT_IMAGE

FROM ${COGMENT_BUILD_ENVIRONMENT_IMAGE}

COPY . .

ENV BUILD_DIR=/workspace/build/linux_amd64
VOLUME /workspace/build/linux_amd64

ENV INSTALL_DIR=/workspace/install/linux_amd64
VOLUME /workspace/install/linux_amd64

CMD ./build_linux.sh

FROM registry.gitlab.com/ai-r/cogment-build-env:orchestrator_protos as build

WORKDIR workspace

COPY . .

RUN cmake --CMAKE_BUILD_TYPE=Release . \
    && make -j$(nproc)               \
    && make install

FROM ubuntu:20.04

COPY --from=build /usr/local/bin/orchestrator /usr/local/bin/

WORKDIR /app

ENTRYPOINT ["orchestrator"]
CMD ["--config=/app/cogment.yaml"]

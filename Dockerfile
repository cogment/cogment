FROM cogment/orchestrator-build-env:v1.0.1 as build

WORKDIR /workspace

COPY . .

RUN mkdir build_dbg && cd build_dbg      \
  && cmake -DCMAKE_BUILD_TYPE=Debug .. \
  && make -j$(nproc)                   \
  && make install

RUN mkdir build_rel && cd build_rel        \
  && cmake -DCMAKE_BUILD_TYPE=Release .. \
  && make -j$(nproc)                     \
  && make install

FROM ubuntu:20.04

COPY --from=build /usr/local/bin/orchestrator /usr/local/bin/
COPY --from=build /usr/local/bin/orchestrator_dbg /usr/local/bin/

WORKDIR /app

ENTRYPOINT ["orchestrator"]
CMD ["--config=/app/cogment.yaml"]

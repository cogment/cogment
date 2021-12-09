FROM cogment/orchestrator-build-env:v2.0.0-debug as debugbuild

WORKDIR /workspace
COPY . .

RUN mkdir build_dbg && cd build_dbg      \
  && cmake -DCMAKE_BUILD_TYPE=Debug .. \
  && make -j$(nproc)                   \
  && make install

FROM cogment/orchestrator-build-env:v2.0.0 as build

WORKDIR /workspace
COPY . .

RUN mkdir build_release && cd build_release \
  && cmake -DCMAKE_BUILD_TYPE=Release ..    \
  && make -j$(nproc)                        \
  && make install


FROM ubuntu:20.04

COPY --from=debugbuild /usr/local/bin/orchestrator_dbg /usr/local/bin/
COPY --from=build      /usr/local/bin/orchestrator     /usr/local/bin/

WORKDIR /app

ENTRYPOINT ["orchestrator"]


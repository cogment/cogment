# Copyright 2023 AI Redefined Inc. <dev+cogment@ai-r.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM ubuntu:20.04

ARG COGMENT_EXEC=./install/linux_amd64/bin/cogment
COPY ${COGMENT_EXEC} /usr/local/bin/cogment
RUN chmod +x /usr/local/bin/cogment

VOLUME /data
ENV COGMENT_MODEL_REGISTRY_ARCHIVE_DIR=/data/model_registry

ENTRYPOINT ["cogment"]

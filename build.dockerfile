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

ARG COGMENT_BUILD_ENVIRONMENT_IMAGE

FROM ${COGMENT_BUILD_ENVIRONMENT_IMAGE}

COPY . .

ENV BUILD_DIR=/workspace/build/linux_amd64
VOLUME /workspace/build/linux_amd64

ENV INSTALL_DIR=/workspace/install/linux_amd64
VOLUME /workspace/install/linux_amd64

CMD ./build_linux.sh

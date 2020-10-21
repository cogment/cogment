// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configurator

const DOCKERFILE = `
FROM python:3.7

# Uncomment following and set specific version if required for only this service
#ENV COGMENT_VERSION 0.3.0a5
#RUN pip install cogment==$COGMENT_VERSION
# Comment following if above is used
ADD requirements.txt .
RUN pip install -r requirements.txt

WORKDIR /app

ADD configurator ./configurator/
ADD cog_settings.py .
ADD *_pb2.py .

CMD ["python", "configurator/main.py"]
`

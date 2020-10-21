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

package js_clients

const PACKAGE_JSON = `
{
  "name": "{{.ProjectName}}",
  "version": "1.0.0",
  "description": "Bootstrap {{.ProjectName}} Project",
  "main": "main.js",
  "scripts": {
    "build": "webpack",
    "start": "webpack-dev-server --host 0.0.0.0 --hot --inline"
  },
  "author": "",
  "license": "ISC",
  "dependencies": {
    "@improbable-eng/grpc-web": "^0.11.0",
    "cogment": "^0.3.0-alpha5",
    "google-protobuf": "^3.7.1",
    "grpc-web": "^1.0.4",
    "ts-protoc-gen": "^0.12.0"
  },
  "devDependencies": {
    "webpack": "^4.41.5",
    "webpack-cli": "^3.3.8",
    "webpack-dev-server": "^3.10.1"
  }
}
`

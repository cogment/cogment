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

package templates

const ROOT_README = `
This is a sample bootstraping project for the Cogment python SDK.

Clone this repository, and you should be good to go.

For an introduction to Cogment or installation instructions see [docs.cogment.ai](https://docs.cogment.ai/).

## Getting started

Before anything else, you should make a copy of the default docker-compose 
overrides. This will allow you to make changes to the project without having to 
rebuild everything all the time.

the docker-compose.override.yml file is not pushed to the repo, so if you want
local changes to the docker-compose setup, that's the place to do it.

	cp docker-compose.override.template.yml docker-compose.override.yml

## Useful commands:

Build the project:

    cogment run build

Launch the environment, agents and orchestrator:

    cogment run start 

Run the client application:

    cogment run client
`

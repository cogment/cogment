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


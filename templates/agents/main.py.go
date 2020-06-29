package agents

const MAIN_PY = `
import cog_settings
from data_pb2 import {{.|pascalify}}Action

from cogment import Agent, GrpcServer

class {{.|pascalify}}(Agent):
    VERSIONS = {"{{.|snakeify}}": "1.0.0"}
    actor_class = cog_settings.actor_classes.{{.|snakeify}}

    def decide(self, observation: cog_settings.actor_classes.{{.|snakeify}}.observation_space):
        print("{{.|pascalify}} decide")
        action = {{.|pascalify}}Action()
        return action

    def reward(self, reward):
        print("{{.|pascalify}} reward")

    def end(self):
        print("{{.|pascalify}} end")


if __name__ == '__main__':
    server = GrpcServer({{.|pascalify}}, cog_settings)
    server.serve()
`

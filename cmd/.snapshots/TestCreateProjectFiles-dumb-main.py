import cog_settings
from data_pb2 import DumbAction

from cogment import Agent, GrpcServer

class Dumb(Agent):
    VERSIONS = {"dumb": "1.0.0"}
    actor_class = cog_settings.actor_classes.dumb

    def decide(self, observation: cog_settings.actor_classes.dumb.observation_space):
        print("Dumb decide")
        action = DumbAction()
        return action

    def reward(self, reward):
        print("Dumb reward")

    def on_message(self, sender, msg):
        if msg:
            print(f'Agent {self.id_in_class} received message - {msg} from sender {sender}')

    def end(self):
        print("Dumb end")


if __name__ == '__main__':
    server = GrpcServer(Dumb, cog_settings)
    server.serve()


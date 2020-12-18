import cog_settings
from data_pb2 import SmartAction

from cogment import Agent, GrpcServer

class Smart(Agent):
    VERSIONS = {"smart": "1.0.0"}
    actor_class = cog_settings.actor_classes.smart

    def decide(self, observation: cog_settings.actor_classes.smart.observation_space):
        print("Smart decide")
        action = SmartAction()
        return action

    def reward(self, reward):
        print("Smart reward")

    def on_message(self, sender, msg):
        if msg:
            print(f'Agent {self.id_in_class} received message - {msg} from sender {sender}')

    def end(self):
        print("Smart end")


if __name__ == '__main__':
    server = GrpcServer(Smart, cog_settings)
    server.serve()


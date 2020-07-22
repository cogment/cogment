package envs

const MAIN_PY = `
import cog_settings
from data_pb2 import Observation

from cogment import Environment, GrpcServer


class Env(Environment):
    VERSIONS = {"env": "1.0.0"}

    def start(self, config):
        print("environment starting")
        observation = Observation()

        # Assign that observation to every single actor.
        obs_table = cog_settings.ObservationsTable(self.trial)
        for o in obs_table.all_observations():
            o.snapshot = observation

        return obs_table

    def update(self, actions: cog_settings.ActionsTable):
        print("environment updating")
        observation = Observation()

        # Assign that observation to every single actor.
        obs_table = cog_settings.ObservationsTable(self.trial)
        for o in obs_table.all_observations():
            o.snapshot = observation

        return obs_table

    def on_message(self, sender, msg):
        if msg:
            print(f'Environment received message - {msg} from sender {sender}')

    def end(self):
        print("environment end")


if __name__ == "__main__":
    server = GrpcServer(Env, cog_settings)
    server.serve()
`

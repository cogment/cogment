

import traceback
from cogment import TrialHooks, GrpcServer
from types import SimpleNamespace as ns

// endpoints
SMART_URL = 'grpc://smart:9000'
DUMB_URL = 'grpc://dumb:9000'


class Supervisor(TrialHooks):
    VERSIONS = {"poker": "1.0.0"}

    @staticmethod
    def pre_trial(trial_id, user_id, trial_params):

        actor_settings = {

            "smart": ns(
            actor_class='smart',
            end_point=SMART_URL,
            config=None
            ),


            "dumb": ns(
            actor_class='dumb',
            end_point=DUMB_URL,
            config=None
            ),


        }


        try:
            trial_config = trial_params.trial_config
            actors = [ns(
                actor_class='master',
                endpoint="human",
                config=None
            )]


            # modify following to retrieve config info and create actors list
            #for ??? in trial_config.env_config.???:
            #    actors.append(actor_settings[???])

            trial_params.actors = actors

            trial_params.environment.config = trial_config.env_config

            return trial_params
        except Exception:
            traceback.print_exc()
            raise

if __name__ == '__main__':
    server = GrpcServer(Configurator, cog_settings)
    server.serve()


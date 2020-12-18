
import cog_settings

from data_pb2 import MasterAction

from cogment.client import Connection

# Callback received message
def handle_messages(sender, msg):
    print(f'Client received message - {msg} from sender {sender}')

# Create a connection to the Orchestrator serving this project
conn = Connection(cog_settings, "orchestrator:9000")

# Initiate a trial
trial = conn.start_trial(cog_settings.actor_classes.master)

# Perform actions, and get observations
observation = trial.do_action(MasterAction(), on_message = handle_messages)
observation = trial.do_action(MasterAction(), on_message = handle_messages)



# cleanup
trial.end()


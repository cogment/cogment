import cog_settings
from data_pb2 import MasterAction
from data_pb2 import TrialConfig

import cogment

import asyncio

async def master_human(actor_session):
    actor_class = cog_settings.actor_classes.master

    actor_session.start()

    async for event in actor_session.event_loop():
        if "observation" in event:
            observation = event["observation"]
            print("Master decide")
            action = MasterAction()
            actor_session.do_action(action)
        if "reward" in event:
            reward = event["reward"]
            print("Master reward")
        if "message" in event:
            (msg, sender) = event["message"]
            print(f"Master received message - {msg} from sender {sender}")

async def main():
    print("Client up and running.")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")

    context.register_actor(
        impl=master_human,
        impl_name="master_human",
        actor_classes=["master"])

    # Create and join a new trial
    trial_id = None
    async def trial_controler(control_session):
        nonlocal trial_id
        print(f"Trial '{control_session.get_trial_id()}' starts")
        trial_id = control_session.get_trial_id()
        # Let the trial play for a while
        await asyncio.sleep(10)
        print(f"Trial '{control_session.get_trial_id()}' terminating")
        await control_session.terminate_trial()

    await context.start_trial(endpoint="orchestrator:9000", impl=trial_controler, trial_config=TrialConfig())

if __name__ == '__main__':
    asyncio.run(main())


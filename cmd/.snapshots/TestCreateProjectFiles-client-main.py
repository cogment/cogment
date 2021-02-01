import cog_settings
from data_pb2 import MasterAction
from data_pb2 import TrialConfig

import cogment

import asyncio

async def human(actor_session):
    actor_session.start()

    async for event in actor_session.event_loop():
        if "observation" in event:
            observation = event["observation"]
            print(f"'{actor_session.name}' received an observation: '{observation}'")
            action = MasterAction()
            actor_session.do_action(action)
        if "reward" in event:
            reward = event["reward"]
            print(f"{actor_session.name} received a reward for tick #{reward.tick_id}: {reward.value}/{reward.confidence}")
        if "message" in event:
            (sender, message) = event["message"]
            print(f"'{actor_session.name}' received a message from '{sender}': - '{message}'")
        if "final_data" in event:
            final_data = event["observation"]
            for observation in final_data.observations:
                print(f"'{actor_session.name}' received a final observation: '{observation}'")
            for reward in final_data.rewards:
                print(f"{actor_session.name} received a final reward for tick #{reward.tick_id}: {reward.value}/{reward.confidence}")
            for message in final_data.messages:
                print(f"'{actor_session.name}' received a final message from '{sender}': - '{message}'")
async def main():
    print("Client starting...")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")
    context.register_actor(
        impl=human,
        impl_name="human",
        actor_classes=["master",])

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


import cog_settings
from data_pb2 import MasterAction
from data_pb2 import TrialConfig

import cogment

import asyncio

async def human(actor_session):
    actor_session.start()

    async for event in actor_session.event_loop():
        if event.observation:
            observation = event.observation
            print(f"'{actor_session.name}' received an observation: '{observation}'")
            if event.type == cogment.EventType.ACTIVE:
                action = MasterAction()
                actor_session.do_action(action)
        for reward in event.rewards:
            print(f"'{actor_session.name}' received a reward for tick #{reward.tick_id}: {reward.value}/{reward.confidence}")
        for message in event.messages:
            print(f"'{actor_session.name}' received a message from '{message.sender_name}': - '{message.payload}'")
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


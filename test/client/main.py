import cog_settings
from data_pb2 import HAction
from data_pb2 import TrialConfig

import cogment

import asyncio

async def client_actor_impl(actor_session):
    actor_session.start()

    async for event in actor_session.event_loop():
        if event.observation:
            observation = event.observation
            print(f"'{actor_session.name}' received an observation: '{observation}'")
            if event.type == cogment.EventType.ACTIVE:
                action = HAction()
                actor_session.do_action(action)
        for reward in event.rewards:
            print(f"'{actor_session.name}' received a reward for tick #{reward.tick_id}: {reward.value}")
        for message in event.messages:
            print(f"'{actor_session.name}' received a message from '{message.sender_name}': - '{message.payload}'")
async def main():
    print("Client starting...")

    context = cogment.Context(cog_settings=cog_settings, user_id="test")
    context.register_actor(
        impl=client_actor_impl,
        impl_name="client_actor_impl",
        actor_classes=["h",])

    # Create a controller
    controller = context.get_controller(endpoint=cogment.Endpoint("orchestrator:9000"))

    # Start a new trial
    trial_id = await controller.start_trial(trial_config=TrialConfig())
    print(f"Trial '{trial_id}' started")

    # Let the trial play for a while
    await asyncio.sleep(10)

    # Terminate the trial
    await controller.terminate_trial(trial_id)
    print(f"Trial '{trial_id}' terminated")

if __name__ == '__main__':
    asyncio.run(main())

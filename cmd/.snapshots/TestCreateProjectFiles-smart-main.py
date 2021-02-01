import cog_settings
from data_pb2 import SmartAction

import cogment

import asyncio

async def smart_impl(actor_session):
    actor_session.start()

    async for event in actor_session.event_loop():
        if "observation" in event:
            observation = event["observation"]
            print(f"'{actor_session.name}' received an observation: '{observation}'")
            action = SmartAction()
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
    print("Smart actor service starting...")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")
    context.register_actor(
        impl=smart_impl,
        impl_name="smart_impl",
        actor_classes=["smart",])

    await context.serve_all_registered(port=9000)

if __name__ == '__main__':
    asyncio.run(main())


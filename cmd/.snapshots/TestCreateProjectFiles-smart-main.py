import cog_settings
from data_pb2 import SmartAction

import cogment

import asyncio

async def smart_impl(actor_session):
    actor_session.start()

    async for event in actor_session.event_loop():
        if event.observation:
            observation = event.observation
            print(f"'{actor_session.name}' received an observation: '{observation}'")
            if event.type == cogment.EventType.ACTIVE:
                action = SmartAction()
                actor_session.do_action(action)
        for reward in event.rewards:
            print(f"'{actor_session.name}' received a reward for tick #{reward.tick_id}: {reward.value}")
        for message in event.messages:
            print(f"'{actor_session.name}' received a message from '{message.sender_name}': - '{message.payload}'")
async def main():
    print("Smart actor service starting...")

    context = cogment.Context(cog_settings=cog_settings, user_id="testit")
    context.register_actor(
        impl=smart_impl,
        impl_name="smart_impl",
        actor_classes=["smart",])

    await context.serve_all_registered(cogment.ServedEndpoint(port=9000))

if __name__ == '__main__':
    asyncio.run(main())


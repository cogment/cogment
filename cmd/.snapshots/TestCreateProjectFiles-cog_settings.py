import cogment as _cog
from types import SimpleNamespace
from typing import List

import data_pb2 as data_pb


protolib = "data_pb2"

_master_class = _cog.actor.ActorClass(
    id="master",
    config_type=None,
    action_space=data_pb.MasterAction,
    observation_space=data_pb.Observation,
    observation_delta=data_pb.Observation,
    observation_delta_apply_fn=_cog.delta_encoding._apply_delta_replace,
    feedback_space=None
)

_smart_class = _cog.actor.ActorClass(
    id="smart",
    config_type=None,
    action_space=data_pb.SmartAction,
    observation_space=data_pb.Observation,
    observation_delta=data_pb.Observation,
    observation_delta_apply_fn=_cog.delta_encoding._apply_delta_replace,
    feedback_space=None
)

_dumb_class = _cog.actor.ActorClass(
    id="dumb",
    config_type=None,
    action_space=data_pb.DumbAction,
    observation_space=data_pb.Observation,
    observation_delta=data_pb.Observation,
    observation_delta_apply_fn=_cog.delta_encoding._apply_delta_replace,
    feedback_space=None
)


actor_classes = _cog.actor.ActorClassList(
    _master_class,
    _smart_class,
    _dumb_class,
)

trial = SimpleNamespace(config_type=None)

# Environment
environment = SimpleNamespace(config_type=data_pb.EnvConfig)


class ActionsTable:
    master: List[data_pb.MasterAction]
    smart: List[data_pb.SmartAction]
    dumb: List[data_pb.DumbAction]

    def __init__(self, trial):
        self.master = [data_pb.MasterAction() for _ in range(trial.actor_counts[0])]
        self.smart = [data_pb.SmartAction() for _ in range(trial.actor_counts[1])]
        self.dumb = [data_pb.DumbAction() for _ in range(trial.actor_counts[2])]

    def all_actions(self):
        return self.master + self.smart + self.dumb



import cogment as _cog
from types import SimpleNamespace
from typing import List

import data_pb2
import subdir.otherdata_pb2
import delta

_plane_class = _cog.ActorClass(
    id='plane',
    config_type=None,
    action_space=data_pb2.Human_PlaneAction,
    observation_space=data_pb2.Observation,
    observation_delta=data_pb2.ObservationDelta,
    observation_delta_apply_fn=delta.apply_delta,
    feedback_space=None
)

_ai_drone_class = _cog.ActorClass(
    id='ai_drone',
    config_type=data_pb2.DroneConfig,
    action_space=data_pb2.Ai_DroneAction,
    observation_space=data_pb2.Observation,
    observation_delta=data_pb2.ObservationDelta,
    observation_delta_apply_fn=delta.apply_delta,
    feedback_space=None
)


actor_classes = _cog.actor_class.ActorClassList(
    _plane_class,
    _ai_drone_class,
)

trial = SimpleNamespace(
    config_type=data_pb2.TrialConfig,
)

# Environment
environment = SimpleNamespace(
    config_type=subdir.otherdata_pb2.Data,
)


class ActionsTable:
    plane: List[data_pb2.Human_PlaneAction]
    ai_drone: List[data_pb2.Ai_DroneAction]

    def __init__(self, trial):
        self.plane = [data_pb2.Human_PlaneAction() for _ in range(trial.actor_counts[0])]
        self.ai_drone = [data_pb2.Ai_DroneAction() for _ in range(trial.actor_counts[1])]

    def all_actions(self):
        return self.plane + self.ai_drone


class plane_ObservationProxy(_cog.env_service.ObservationProxy):
    @property
    def snapshot(self) -> data_pb2.Observation:
        return self._get_snapshot(data_pb2.Observation)

    @snapshot.setter
    def snapshot(self, v):
        self._set_snapshot(v)

    @property
    def delta(self) -> data_pb2.ObservationDelta:
        return self._get_delta(data_pb2.ObservationDelta)

    @delta.setter
    def delta(self, v):
        self._set_delta(v)

class ai_drone_ObservationProxy(_cog.env_service.ObservationProxy):
    @property
    def snapshot(self) -> data_pb2.Observation:
        return self._get_snapshot(data_pb2.Observation)

    @snapshot.setter
    def snapshot(self, v):
        self._set_snapshot(v)

    @property
    def delta(self) -> data_pb2.ObservationDelta:
        return self._get_delta(data_pb2.ObservationDelta)

    @delta.setter
    def delta(self, v):
        self._set_delta(v)


class ObservationsTable:
    plane: List[plane_ObservationProxy]
    ai_drone: List[ai_drone_ObservationProxy]

    def __init__(self, trial):
        self.plane = [plane_ObservationProxy() for _ in range(trial.actor_counts[0])]
        self.ai_drone = [ai_drone_ObservationProxy() for _ in range(trial.actor_counts[1])]

    def all_observations(self):
        return self.plane + self.ai_drone

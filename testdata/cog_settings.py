# Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


from types import SimpleNamespace
from typing import List

import cogment as _cog
import data_pb2
import subdir.otherdata_pb2

_plane_class = _cog.ActorClass(
    id='plane',
    config_type=None,
    action_space=data_pb2.Human_PlaneAction,
    observation_space=data_pb2.Observation,
)

_ai_drone_class = _cog.ActorClass(
    id='ai_drone',
    config_type=data_pb2.DroneConfig,
    action_space=data_pb2.Ai_DroneAction,
    observation_space=data_pb2.Observation,
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
        self.plane = [data_pb2.Human_PlaneAction()
                      for _ in range(trial.actor_counts[0])]
        self.ai_drone = [data_pb2.Ai_DroneAction()
                         for _ in range(trial.actor_counts[1])]

    def all_actions(self):
        return self.plane + self.ai_drone


class plane_ObservationProxy(_cog.env_service.ObservationProxy):
    @property
    def snapshot(self) -> data_pb2.Observation:
        return self._get_snapshot(data_pb2.Observation)

    @snapshot.setter
    def snapshot(self, v):
        self._set_snapshot(v)


class ai_drone_ObservationProxy(_cog.env_service.ObservationProxy):
    @property
    def snapshot(self) -> data_pb2.Observation:
        return self._get_snapshot(data_pb2.Observation)

    @snapshot.setter
    def snapshot(self, v):
        self._set_snapshot(v)


class ObservationsTable:
    plane: List[plane_ObservationProxy]
    ai_drone: List[ai_drone_ObservationProxy]

    def __init__(self, trial):
        self.plane = [plane_ObservationProxy()
                      for _ in range(trial.actor_counts[0])]
        self.ai_drone = [ai_drone_ObservationProxy()
                         for _ in range(trial.actor_counts[1])]

    def all_observations(self):
        return self.plane + self.ai_drone

import * as data_pb2 from './data_pb.js';
import * as delta from './delta.js';


const _plane_class = {
    name: 'plane',
    action_space: data_pb2.Human_PlaneAction,
    observation_space: data_pb2.Observation,
    observation_delta: data_pb2.ObservationDelta,
    observation_delta_apply_fn: delta.apply_delta,
};

const _ai_drone_class = {
    name: 'ai_drone',
    action_space: data_pb2.Ai_DroneAction,
    observation_space: data_pb2.Observation,
    observation_delta: data_pb2.ObservationDelta,
    observation_delta_apply_fn: delta.apply_delta,
};


const settings = {
    actor_classes: {
        plane: _plane_class,
        ai_drone: _ai_drone_class,
    },

    trial: {
        config: data_pb2.TrialConfig,
    },

    environment: {
        config: null,
    }
};

export default settings;

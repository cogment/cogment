import * as data_pb2 from './data_pb.js';


const _plane_class = {
    name: 'plane',
    action_space: data_pb2.Human_PlaneAction,
    observation_space: data_pb2.Observation,
};

const _ai_drone_class = {
    name: 'ai_drone',
    action_space: data_pb2.Ai_DroneAction,
    observation_space: data_pb2.Observation,
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

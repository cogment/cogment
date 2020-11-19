// Copyright 2020 Artificial Intelligence Redefined <dev+cogment@ai-r.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package replaybuffer

const REPLAYBUFFER_PY = `
import logging
from distutils.util import strtobool
import os
from concurrent import futures
from cogment.api.data_pb2 import LogExporterSampleReply, _LOGEXPORTER
from cogment.api.data_pb2_grpc import LogExporterServicer, add_LogExporterServicer_to_server
import grpc
from redis import Redis
from grpc_reflection.v1alpha import reflection


ENABLE_REFLECTION_VAR_NAME = 'COGMENT_GRPC_REFLECTION'


REDIS_CONNECT = Redis(host='redis', port=6379)


class ReplayBuffer(LogExporterServicer):
    """ Implements log exporter service for deathmatch
    """

    def __init__(self):
        self._logger = logging.getLogger(__file__)
        self.trial_params = ""

    # pylint: disable=invalid-name
    # pylint: disable=broad-except
    # pylint: disable=unused-argument
    def OnLogSample(self, request_iterator, context):
        store_pipe = REDIS_CONNECT.pipeline()

        try:
            for request in request_iterator:
                if request.HasField("trial_params"):
                    self.trial_params = request.trial_params.SerializeToString()

                elif request.HasField("sample"):
                    key = f"{request.sample.trial_id}_{request.sample.observations.tick_id}"

                    data_point = request.sample.SerializeToString()
                    store_pipe.hset(key, "trial_params", self.trial_params)

                    store_pipe.hset(key, "sample", data_point)
                    store_pipe.rpush("last_trial_keys", key)
            store_pipe.execute()
        except Exception as e:
            self._logger.warning(f"{e}")

        return LogExporterSampleReply()


def serve():

    server = grpc.server(futures.ThreadPoolExecutor(thread_name_prefix="log_exporter"))
    add_LogExporterServicer_to_server(ReplayBuffer(), server)

    # Enable grpc reflection if requested
    if strtobool(os.getenv(ENABLE_REFLECTION_VAR_NAME, 'false')):
        service_names = (_LOGEXPORTER.full_name, reflection.SERVICE_NAME,)
        reflection.enable_server_reflection(service_names, server)

    server.add_insecure_port('[::]:9000')
    server.start()
    logging.info('server started on port 9000')
    server.wait_for_termination()


if __name__ == '__main__':
    serve()
`

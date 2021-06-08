// Copyright 2021 AI Redefined Inc. <dev+cogment@ai-r.com>
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

#ifndef NDEBUG
  #define SPDLOG_ACTIVE_LEVEL SPDLOG_LEVEL_TRACE
#endif

#include "cogment/utils.h"

#ifdef __linux__
  #include <string.h>
  #include <time.h>

// This is simpler and much more efficient than the C++ "chrono" way
uint64_t Timestamp() {
  struct timespec ts;
  int res = clock_gettime(CLOCK_REALTIME, &ts);
  if (res == -1) {
    throw MakeException("Could not get time stamp: %s", strerror(errno));
  }

  static constexpr uint64_t NANOS = 1'000'000'000;
  return (ts.tv_sec * NANOS + ts.tv_nsec);
}

#endif

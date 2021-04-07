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

#ifndef COGMENT_UTILS_H_INCLUDED
#define COGMENT_UTILS_H_INCLUDED

#include <cstdarg>
#include <cstdio>
#include "easy_grpc/easy_grpc.h"
#include "spdlog/spdlog.h"

namespace cogment {
template <typename T>
using expected = aom::expected<T>;

template <typename T>
using Future = aom::Future<T>;

template <typename T>
using Promise = aom::Promise<T>;
}  // namespace cogment

template <class EXC = std::runtime_error>
EXC MakeException(const char* format, ...) {
  static constexpr std::size_t BUF_SIZE = 256;
  char buf[BUF_SIZE];

  va_list args;
  va_start(args, format);
  std::vsnprintf(buf, BUF_SIZE, format, args);
  va_end(args);

  const char* const const_buf = buf;
  spdlog::error("**Exception generated**: {}", const_buf);
  return EXC(const_buf);
}

#endif

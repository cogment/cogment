#ifndef COGMENT_UTILS_H_INCLUDED
#define COGMENT_UTILS_H_INCLUDED

#include "easy_grpc/easy_grpc.h"

namespace cogment {
template <typename T>
using expected = aom::expected<T>;

template <typename T>
using Future = aom::Future<T>;

template <typename T>
using Promise = aom::Promise<T>;
}  // namespace cogment
#endif
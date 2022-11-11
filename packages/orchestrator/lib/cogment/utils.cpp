// Copyright 2022 AI Redefined Inc. <dev+cogment@ai-r.com>
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
#include <fstream>
#include <cstdlib>

namespace cogment {

#if defined(__linux__) || defined(__APPLE__)
  #include <string.h>
  #include <time.h>

// This is simpler and much more efficient than the C++ "std::chrono" way
uint64_t Timestamp() {
  struct timespec ts;
  int res = clock_gettime(CLOCK_REALTIME, &ts);
  if (res != -1) {
    return (ts.tv_sec * NANOS + ts.tv_nsec);
  }
  else {
    throw MakeException("Could not get time stamp: [{}]", strerror(errno));
  }
}
#else
uint64_t Timestamp() {
  // If assert fails, need to cast to nanoseconds -> chrono::time_point_cast
  static_assert(std::is_same_v<std::chrono::high_resolution_clock::duration, std::chrono::nanoseconds>);

  const auto now = std::chrono::high_resolution_clock::now();
  return now.time_since_epoch().count();
}
#endif

#if defined(__linux__) || defined(__APPLE__)
  #if defined(__APPLE__)
    #warning "Despite documentation, the 'h_errno' variable is not found by the linker (-lc)"
int h_errno;
  #endif

  #include <string.h>
  #include <unistd.h>
  #include <netdb.h>
  #include <arpa/inet.h>

  #define ERROR_CODE1 strerror(errno)
  #define ERROR_CODE2 hstrerror(h_errno)

static void PreGetHostAddress() {}
static void PostGetHostAddress() {}

#elif defined(_WIN64)
  #include <winsock.h>

  #define ERROR_CODE1 WSAGetLastError()
  #define ERROR_CODE2 WSAGetLastError()

static void PreGetHostAddress() {
  WSADATA data;
  const int res = WSAStartup(MAKEWORD(2, 2), &data);
  if (res != 0) {
    throw MakeException("Could not initialise winsock [{}] [{}]", res, WSAGetLastError());
  }
}

static void PostGetHostAddress() { WSACleanup(); }

#else
  #error "Unknown target OS"
#endif

#if defined(__linux__)

NetChecker::NetChecker() {
  parse_ip_procfs("/proc/net/tcp");
  parse_ip_procfs("/proc/net/tcp6");
}

void NetChecker::parse_ip_procfs(const std::string& path) {
  try {
    std::ifstream fs(path);
    if (!fs.is_open()) {
      spdlog::warn("Failed to open IP state [{}]", path);
      return;
    }

    // Discarding first line (header)
    std::string header;
    getline(fs, header);
    // Header: "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode"
    SPDLOG_DEBUG("[{}]: {}", path, header);

    for (std::string line; getline(fs, line).good();) {
      if (line.empty()) {
        continue;
      }
      SPDLOG_DEBUG("[{}]: {}", path, line);

      // Format is fixed, e.g.: "   5: 0341A8C0:A0C2 0141A8C0:0C38 06 00000000:00000000 03:000000D9 00000000 ..."
      auto columns = split(trim(line), ' ', true);
      if (columns.size() < 2) {
        spdlog::warn("Unexpected IP state line format [{}]", line);
        continue;
      }
      auto address_port = split(columns[1], ':');
      if (address_port.size() < 2) {
        spdlog::warn("Unexpected IP state address format [{}] in [{}]", columns[1], line);
        continue;
      }
      auto port = address_port[1];
      if (port.length() != 4) {
        spdlog::warn("Invalid port format in IP state [{}] [{}]", port, line);
        continue;
      }

      std::string port_str(port);
      auto port_int = std::strtoull(port_str.c_str(), nullptr, 16);
      if (port_int == 0 || port_int > std::numeric_limits<uint16_t>::max()) {
        spdlog::warn("Invalid port found in IP state [{}]", port_str);
        continue;
      }
      SPDLOG_DEBUG("Port found [{}]", port_int);

      m_tcp_ports.insert(port_int);
    }

    if (!fs.eof()) {
      spdlog::warn("Failure reading IP state");
    }
  }
  catch (const std::exception& exc) {
    spdlog::warn("Failed to process IP state [{}]", exc.what());
    m_tcp_ports.clear();
  }
  catch (...) {
    spdlog::warn("Failed to process IP state");
    m_tcp_ports.clear();
  }
}
#else
  #pragma message("NetChecker is not implemented for target OS")
// Windows could use 'GetTcpTable'
NetChecker::NetChecker() {}
#endif

bool NetChecker::is_tcp_port_used(uint16_t port) { return (m_tcp_ports.find(port) != m_tcp_ports.end()); }

std::string GetHostAddress() {
  PreGetHostAddress();
  static constexpr size_t MAX_HOST_NAME_LENGTH = 256;
  char host_name[MAX_HOST_NAME_LENGTH];
  const int res = gethostname(host_name, MAX_HOST_NAME_LENGTH - 1);
  if (res != 0) {
    throw MakeException("Could not get host name: [{}] [{}]", res, ERROR_CODE1);
  }
  host_name[MAX_HOST_NAME_LENGTH - 1] = '\0';
  spdlog::debug("Host name [{}]", host_name);

  const struct hostent* host_data = gethostbyname(host_name);
  if (host_data == NULL) {
    throw MakeException("Could not get address for hostname [{}]: [{}]", host_name, ERROR_CODE2);
  }

  if (host_data->h_length <= 0) {
    throw MakeException("No address for hostname [{}]", host_name);
  }

  char** const addr_list = host_data->h_addr_list;
  if (addr_list == NULL || addr_list[0] == NULL) {
    throw MakeException("Null address for hostname [{}]", host_name);
  }

  const char* ip_address = inet_ntoa(*((struct in_addr*)addr_list[0]));
  if (ip_address == NULL) {
    throw MakeException("Could not translate address for hostname [{}]", host_name);
  }

  PostGetHostAddress();
  return ip_address;
}

std::vector<std::string_view> split(std::string_view in, char separator, bool merge_separators) {
  std::vector<std::string_view> result;

  while (true) {
    const size_t sep_index = in.find(separator);
    if (sep_index != in.npos) {
      size_t token_index = sep_index + 1;

      if (merge_separators) {
        for (; token_index < in.length(); token_index++) {
          if (in[token_index] != separator) {
            break;
          }
        }
      }

      result.emplace_back(in.substr(0, sep_index));
      in.remove_prefix(token_index);
    }
    else {
      result.emplace_back(in);
      break;
    }
  }

  return result;
}

std::string_view trim(std::string_view in) {
  std::string_view result = in;

  for (auto val : in) {
    if (!std::isspace(static_cast<unsigned char>(val))) {
      break;
    }
    else {
      result.remove_prefix(1);
    }
  }

  for (auto itor = in.rbegin(); itor != in.rend(); itor++) {
    const auto val = static_cast<unsigned char>(*itor);
    if (!std::isspace(val)) {
      break;
    }
    else {
      result.remove_suffix(1);
    }
  }

  return result;
}

std::vector<PropertyView> parse_properties(std::string_view in, char property_separator, char value_separator) {
  std::vector<PropertyView> result;

  while (!in.empty()) {
    std::string_view one_prop;
    const size_t prop_index = in.find(property_separator);
    if (prop_index != in.npos) {
      one_prop = in.substr(0, prop_index);
      in.remove_prefix(prop_index + 1);
    }
    else {
      one_prop = in;
      in.remove_prefix(in.size());
    }

    const size_t val_index = one_prop.find(value_separator);
    if (val_index != one_prop.npos) {
      auto prop_name = one_prop.substr(0, val_index);
      auto prop_val = one_prop.substr(val_index + 1);
      result.emplace_back(prop_name, prop_val);
    }
    else {
      result.emplace_back(one_prop, std::string_view());
    }
  }

  return result;
}

class ThreadPool::ThreadControl {
public:
  bool is_available() const { return (!m_executing_func && m_active); }

  void stop() {
    m_active = false;  // To stop once func is done
    m_funcs.push({});  // To stop if there is no func running
  }

  std::future<void> set_execution(FUNC_TYPE&& func, std::string_view desc) {
    if (!func) {
      throw MakeException("Non-callable function for thread pool [{}]", desc);
    }

    m_executing_func = true;
    m_prom = std::promise<void>();
    m_description.assign(desc.data(), desc.size());
    m_funcs.push(std::move(func));
    return m_prom.get_future();
  }

  void run() {
    m_active = true;

    try {
      while (m_active) {
        auto func = m_funcs.pop();
        if (!func) {
          break;
        }

        try {
          func();
        }
        catch (const std::exception& exc) {
          spdlog::error("Threaded function [{}] failed [{}]", m_description, exc.what());
        }
        catch (...) {
          spdlog::error("Threaded function [{}] failed", m_description);
        }
        m_prom.set_value();  // No need for 'set_exception' for now
        m_executing_func = false;
      }
    }
    catch (const std::exception& exc) {
      spdlog::error("Pooled thread failure: [{}]", exc.what());
    }
    catch (...) {
      spdlog::error("Pooled thread failure");
    }

    m_active = false;
    m_executing_func = false;
  }

private:
  ThrQueue<FUNC_TYPE> m_funcs;
  bool m_active = false;
  bool m_executing_func = false;
  std::promise<void> m_prom;
  std::string m_description;
};

ThreadPool::~ThreadPool() {
  for (auto& ctrl : m_thread_controls) {
    ctrl->stop();
  }
  for (auto& thr : m_thread_pool) {
    thr.detach();
  }
}

std::future<void> ThreadPool::push(std::string_view desc, FUNC_TYPE&& func) {
  const std::lock_guard lg(m_push_lock);

  for (auto& control : m_thread_controls) {
    if (control->is_available()) {
      return control->set_execution(std::move(func), desc);
    }
  }

  auto& thr_control = add_thread();
  return thr_control->set_execution(std::move(func), desc);
}

std::shared_ptr<ThreadPool::ThreadControl>& ThreadPool::add_thread() {
  m_thread_controls.emplace_back(std::make_shared<ThreadControl>());
  auto& thr_control = m_thread_controls.back();
  m_thread_pool.emplace_back([thr_control]() {
    thr_control->run();
    spdlog::debug("Threadpool thread exiting");
  });

  spdlog::debug("Nb of threads in pool: [{}]", m_thread_controls.size());
  return thr_control;
}

Watchdog::Watchdog(ThreadPool* pool) : m_base_time(std::chrono::steady_clock::now()), m_sec_count(0), m_running(true) {
  m_time_thr = pool->push("Watchdog", [this]() {
    while (m_running) {
      const auto next_count = m_base_time + std::chrono::seconds(m_sec_count + RESOLUTION);
      std::this_thread::sleep_until(next_count);
      m_sec_count++;

      // Assuming it will take less than the resolution to execute!
      process_timeouts(m_sec_count);
    }
  });

  m_execution_thr = pool->push("Watchdog callback execution", [this]() {
    while (m_running) {
      execute_functions();
    }
  });
}

Watchdog::~Watchdog() {
  m_running = false;
  m_execution_queue.push({});
  m_time_thr.wait();
  m_execution_thr.wait();
}

void Watchdog::push(uint16_t timeout_sec, bool auto_repeat, FUNC_TYPE&& func) {
  if (!func) {
    MakeException("Trying to set a watchdog function that is undefined");
  }

  if (timeout_sec != 0) {
    const std::lock_guard lg(m_timed_queue_lock);
    const uint64_t time_count = m_sec_count + timeout_sec;

    if (auto_repeat) {
      m_time_queue.emplace(time_count, timeout_sec, std::move(func));
    }
    else {
      m_time_queue.emplace(time_count, 0, std::move(func));
    }
  }
  else {
    MakeException("Cannot set a watchdog without proper timeout (>0)");
  }
}

void Watchdog::process_timeouts(uint64_t at_sec_count) {
  const std::lock_guard lg(m_timed_queue_lock);

  while (!m_time_queue.empty()) {
    if (m_time_queue.top().entry->sec_count > at_sec_count) {
      break;
    }
    else {
      auto entry_proxy = m_time_queue.top();
      m_time_queue.pop();

      if (!entry_proxy.entry->enabled) {
        SPDLOG_TRACE("The disabled watchdog task was removed");
        continue;
      }

      m_execution_queue.push(entry_proxy.entry);

      const auto next_timeout = entry_proxy.entry->next_timeout;
      if (next_timeout != 0) {
        entry_proxy.entry->sec_count = at_sec_count + next_timeout;
        m_time_queue.emplace(entry_proxy);
      }
    }
  }
}

void Watchdog::execute_functions() {
  auto entry = m_execution_queue.pop();
  if (entry == nullptr || !entry->enabled) {
    return;
  }

  SPDLOG_TRACE("Watchdog task execution at [{}] sec",
               std::chrono::steady_clock::now().time_since_epoch().count() * NANOS_INV);

  bool continue_repeat = false;
  try {
    continue_repeat = entry->func();
  }
  catch (const std::exception& exc) {
    spdlog::error("Watchdog function failed [{}]", exc.what());
  }
  catch (...) {
    spdlog::error("Watchdog function failed");
  }

  if (!continue_repeat) {
    SPDLOG_TRACE("Watchdog function returned false; disabling task");
    entry->enabled = false;
  }
}

MultiWait::MultiWait() : m_waiting(false) {}

void MultiWait::push_back(float timeout_sec, FUT_TYPE&& fut, std::string_view descr) {
  if (m_waiting) {
    throw MakeException("Cannot add futures while waiting");
  }
  if (!fut.valid()) {
    throw MakeException("Cannot add invalid futures");
  }

  const int64_t timeout_ns = timeout_sec * NANOS;
  const size_t fut_index = m_futures.size();
  m_futures.emplace_back(std::move(fut));
  m_wait_queue.emplace(timeout_ns, fut_index, descr);
}

std::vector<size_t> MultiWait::wait_for_all() {
  static const auto zero = std::chrono::nanoseconds::zero();

  SPDLOG_TRACE("starting to wait");

  m_waiting = true;
  const auto base_time = std::chrono::steady_clock::now();
  std::vector<size_t> failed_timed_out;

  for (; !m_wait_queue.empty(); m_wait_queue.pop()) {
    auto& top = m_wait_queue.top();
    auto& fut = m_futures[top.fut_index];

    if (top.wait_time_ns < 0) {
      SPDLOG_TRACE("Waiting indefinitely for [{}]", top.description);
      const auto ready = fut.get();

      if (!ready) {
        failed_timed_out.emplace_back(top.fut_index);
      }

      SPDLOG_TRACE("[{}] seen ready [{}] after [{}] sec", top.description, ready,
                   (std::chrono::steady_clock::now() - base_time).count() * NANOS_INV);
    }
    else {
      SPDLOG_TRACE("Timed waiting for [{}]", top.description);

      const auto wait_time = std::chrono::nanoseconds(top.wait_time_ns);
      const auto now = std::chrono::steady_clock::now();
      auto more_wait = wait_time - (now - base_time);
      if (more_wait < zero) {
        more_wait = zero;
      }

      const auto fut_res = fut.wait_for(more_wait);

      switch (fut_res) {
      case std::future_status::deferred: {
        throw MakeException("Cannot accept deferred action for [{}] in MultiWait", top.description);
        break;
      }
      case std::future_status::ready: {
        const auto ready = fut.get();
        if (!ready) {
          failed_timed_out.emplace_back(top.fut_index);
        }
        SPDLOG_TRACE("[{}] (with timeout [{}]) seen ready [{}] after [{}] sec", top.description,
                     top.wait_time_ns * NANOS_INV, ready,
                     (std::chrono::steady_clock::now() - base_time).count() * NANOS_INV);
        break;
      }
      case std::future_status::timeout: {
        failed_timed_out.emplace_back(top.fut_index);
        SPDLOG_TRACE("[{}] (with timeout [{}]) seen timed out after [{}] sec", top.description,
                     top.wait_time_ns * NANOS_INV, (std::chrono::steady_clock::now() - base_time).count() * NANOS_INV);
        break;
      }
      default: {
        throw MakeException("Failed to wait for [{}] in MultiWait: [{}]", top.description, static_cast<int>(fut_res));
        break;
      }
      }
    }
  }

  m_futures.clear();
  m_waiting = false;

  return failed_timed_out;
}

TimeoutRunner::TimeoutRunner(ThreadPool* pool) : m_active(true), m_sorted(false), m_running_size(0), m_cond_sig(false) {
  m_timeout_thr = pool->push("TimeoutRunner", [this]() {
    while (m_active) {
      SPDLOG_TRACE("TimeoutRunner going to sleep at [{}]",
                   (std::chrono::steady_clock::now() - m_start_time).count() * NANOS_INV);
      wait_next();
      SPDLOG_TRACE("TimeoutRunner wake up at [{}]",
                   (std::chrono::steady_clock::now() - m_start_time).count() * NANOS_INV);
    }
  });
}

TimeoutRunner::~TimeoutRunner() {
  m_active = false;
  {
    const std::lock_guard lg(m_cond_lock);
    m_cond_sig = true;
    m_cond.notify_one();
  }
  m_timeout_thr.wait();
}

void TimeoutRunner::add(float timeout_sec, uint32_t id) {
  const std::lock_guard lg(m_cond_lock);

  static constexpr float TO_NANOS = static_cast<float>(NANOS);
  const uint64_t timeout_nanos = static_cast<uint64_t>(timeout_sec * TO_NANOS);
  const std::chrono::nanoseconds timeout(timeout_nanos);
  m_timeouts.emplace_back(timeout, id);

  m_sorted = false;
}

void TimeoutRunner::remove(uint32_t id) {
  const std::lock_guard lg(m_cond_lock);
  m_timeouts_to_remove.emplace_back(id);

  m_sorted = false;
}

void TimeoutRunner::start(std::function<void()> prep_func) {
  const std::lock_guard lg(m_cond_lock);

  // This is functionality that needs to be done before the first signal,
  // the only way to enforce that is to do it inside the lock.
  prep_func();

  m_start_time = std::chrono::steady_clock::now();

  // This should happen rarely
  if (!m_sorted) {
    if (!m_sig_func || !m_term_func) {
      throw MakeException("The 'sig' and 'term' functions are not defined in TimeoutRunner");
    }

    for (size_t id : m_timeouts_to_remove) {
      for (auto itor = m_timeouts.begin(); itor != m_timeouts.end(); ++itor) {
        if (itor->id == id) {
          m_timeouts.erase(itor);
          break;
        }
      }
    }
    m_timeouts_to_remove.clear();

    std::sort(m_timeouts.begin(), m_timeouts.end());
    m_sorted = true;
  }

  for (auto& entry : m_timeouts) {
    entry.signaled = false;
  }
  m_running_size = m_timeouts.size();

  m_cond_sig = true;
  m_cond.notify_one();

  SPDLOG_TRACE("TimeoutRunner started at [{}] with [{}] entries",
               (std::chrono::steady_clock::now() - m_start_time).count() * NANOS_INV, m_running_size);
}

// 1- Call sig func (increment nb actions) and call term func (check if all actions) if index not in list
// 2- If index in the list, call sig func
//     2.1 - If no more in list, call term func
// 3- When timeout comes to term, call sig func for all non-signaled items, then call term func
// 4- Protect from signaling once timeout comes to term.
void TimeoutRunner::signal(uint32_t id) {
  const std::lock_guard lg(m_cond_lock);
  SPDLOG_TRACE("TimeoutRunner signal caleld for [{}] at [{}] ([{}] entries)", id,
               (std::chrono::steady_clock::now() - m_start_time).count() * NANOS_INV, m_running_size);

  size_t id_size;
  for (id_size = m_running_size; id_size > 0; id_size--) {
    auto& entry = m_timeouts[id_size - 1];
    if (entry.id == id) {
      entry.signaled = true;
      m_sig_func();
      break;
    }
  }

  // Not found or list empty
  if (id_size == 0) {
    m_sig_func();
    m_term_func();
    return;
  }

  // If back of list: shorten list and update timeout
  if (id_size == m_running_size) {
    while (m_running_size > 0 && m_timeouts[m_running_size - 1].signaled) {
      m_running_size--;
    }

    m_cond_sig = true;
    m_cond.notify_one();
  }
}

void TimeoutRunner::wait_next() {
  std::unique_lock ul(m_cond_lock);

  bool signaled;  // If not signaled -> timed out
  if (m_running_size > 0) {
    const auto& top = m_timeouts[m_running_size - 1];
    const auto next_time = m_start_time + top.timeout;
    SPDLOG_TRACE("TimeoutRunner next timeout at [{}]", top.timeout.count() * NANOS_INV);

    signaled = m_cond.wait_until(ul, next_time, [this]() {
      return m_cond_sig;
    });
  }
  else {
    m_cond.wait(ul, [this]() {
      return m_cond_sig;
    });
    signaled = true;
  }

  do_next(signaled);

  m_cond_sig = false;
}

void TimeoutRunner::do_next(bool signaled) {
  if (!signaled) {  // Timed out
    for (; m_running_size > 0; m_running_size--) {
      if (!m_timeouts[m_running_size - 1].signaled) {
        m_sig_func();
      }
    }
  }

  if (m_running_size == 0) {
    m_term_func();
  }
}

}  // namespace cogment

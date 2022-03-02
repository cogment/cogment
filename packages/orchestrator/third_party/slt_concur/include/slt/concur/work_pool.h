
#ifndef SLT_CONCUR_WORK_POOL_H
#define SLT_CONCUR_WORK_POOL_H
#include <functional>
#include <future>
#include <thread>

#include "slt/concur/job_queue.h"

namespace slt {
namespace concur {
class Work_pool;
// Announced jobs interact primarely Work_pool::wait_idle().
// Specifically, this is usefull when dealing with a sequence of jobs accross
// multiple work pools. (as the final pool could appear empty until preceding
// work has been completd.)
class Announced_job {
  friend class Work_pool;

 public:
  using job_type = std::function<void()>;
  
  Announced_job(Announced_job&&);
  Announced_job& operator=(Announced_job&&);
  Announced_job(const Announced_job&);
  Announced_job& operator=(const Announced_job&);
  ~Announced_job();
  void push(job_type job);

 private:
  Announced_job(Work_pool* pool);
  Work_pool* pool_ = nullptr;
};

class Work_pool {
 public:
  using job_type = std::function<void()>;

  Work_pool(std::size_t min_workers, std::size_t max_workers);
  ~Work_pool();

  // If The current thread is a worker, this will return the pool associated
  // with said worker. This is primarily used as a default target.
  static Work_pool* current();

  // Makes this pool the default pool in the curret thread,
  void make_current();

  // Adds work to the pool
  void push_job(job_type job);

  // Announce that a job may be added to the pool. the pool's wait_idle
  // will block until the announced job has been performed or cancelled.
  Announced_job announce_job() { return Announced_job(this); }
  
  // Blocks until the pool is empty of queued and announced jobs.
  void wait_idle();

 private:
  void worker_main();
  int job_done_();
  std::mutex mutex_;
  std::size_t min_workers_;
  std::size_t max_workers_;
  std::condition_variable idle_var;
  std::atomic<int> queued_jobs;
  Job_queue work_;
  std::vector<std::thread> workers_;
  friend class Announced_job;
};
}  // namespace concur
}  // namespace slt
#endif
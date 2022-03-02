#include "slt/concur/work_pool.h"

#include <cassert>

namespace slt {

namespace concur {

namespace {
thread_local Work_pool* current_pool = nullptr;
}

Announced_job::Announced_job(Work_pool* pool) : pool_(pool) {
  pool_->queued_jobs++;
}

Announced_job::Announced_job(Announced_job&& rhs) {
  pool_ = rhs.pool_;
  rhs.pool_ = nullptr;
}

Announced_job::Announced_job(const Announced_job&) {
  assert(false);
}

Announced_job& Announced_job::operator=(const Announced_job&){
  assert(false);
  return *this;
}

Announced_job& Announced_job::operator=(Announced_job&& rhs) {
  pool_ = rhs.pool_;
  rhs.pool_ = nullptr;
  return *this;
}

Announced_job::~Announced_job() {
  if (pool_) {
    pool_->job_done_();
  }
}

void Announced_job::push(job_type job) {
  assert(pool_);

  pool_->push_job(std::move(job));
  pool_->job_done_();
  pool_ = nullptr;
}

Work_pool* Work_pool::current() {
  return current_pool;
}

Work_pool::Work_pool(std::size_t min_workers, std::size_t max_workers)
    : min_workers_(min_workers), max_workers_(max_workers), queued_jobs(0) {
  for (std::size_t i = 0; i < min_workers; ++i) {
    workers_.emplace_back(&Work_pool::worker_main, this);
  }
}

Work_pool::~Work_pool() {
  for (auto& _ : workers_) {
    (void)_;
    work_.push(nullptr);
  }

  for (auto& worker : workers_) {
    worker.join();
  }
}

void Work_pool::push_job(job_type job) {
  assert(job);
  queued_jobs++;
  auto count = work_.push(std::move(job));

  std::lock_guard l(mutex_);
  // If unclaimed work is piling up, create more threads.
  if (count > workers_.size() && workers_.size() < max_workers_) {
    workers_.emplace_back(&Work_pool::worker_main, this);
  }
}

void Work_pool::make_current() {
  current_pool = this;
}

void Work_pool::worker_main() {
  make_current();

  while (1) {
    auto op = work_.pop();
    if (!op) {
      break;
    }

    op();

    job_done_();
  }
}

int Work_pool::job_done_() {
  std::lock_guard l(mutex_);
  auto j = --queued_jobs;
  if (j == 0) {
    idle_var.notify_all();
  }

  return j;
}

void Work_pool::wait_idle() {
  std::unique_lock<std::mutex> l(mutex_);
  idle_var.wait(l, [this] { return queued_jobs == 0; });
}
}  // namespace concur
}  // namespace slt

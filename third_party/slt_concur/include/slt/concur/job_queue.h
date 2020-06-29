#ifndef SLT_CONCUR_JOB_QUEUE_H
#define SLT_CONCUR_JOB_QUEUE_H

#include <queue>
#include <thread>
#include <mutex>
#include <functional>
#include <condition_variable>

namespace slt {
namespace concur {
  class Job_queue {
    public:
      using job_type = std::function<void()>; 
    
      std::size_t push(job_type item) {
        std::unique_lock<std::mutex> lock(mutex_);

        queue_.emplace(std::move(item));
        std::size_t result = queue_.size();

        lock.unlock();
        cond_.notify_one();

        return result;
      }

    job_type pop() {
      std::unique_lock<std::mutex> lock(mutex_);
      cond_.wait(lock, [this]{return !queue_.empty();});
      
      auto item = std::move(queue_.front());
      queue_.pop();
      return item;
    }

    private:
      std::queue<job_type> queue_;
      std::mutex mutex_;
      std::condition_variable cond_;

  };

}  // namespace concur
}  // namespace slt

#endif

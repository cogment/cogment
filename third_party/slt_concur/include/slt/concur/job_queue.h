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

        m_queue.emplace(std::move(item));
        std::size_t result = m_queue.size();

        lock.unlock();
        cond_.notify_one();

        return result;
      }

    job_type pop() {
      std::unique_lock<std::mutex> lock(mutex_);
      cond_.wait(lock, [this]{return !m_queue.empty();});
      
      auto item = std::move(m_queue.front());
      m_queue.pop();
      return item;
    }

    private:
      std::queue<job_type> m_queue;
      std::mutex mutex_;
      std::condition_variable cond_;

  };

}  // namespace concur
}  // namespace slt

#endif

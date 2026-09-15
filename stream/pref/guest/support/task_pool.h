#ifndef TASK_POOL_H
#define TASK_POOL_H

#include <functional>
#include <future>
#include <utility>

// Official preflate starts worker threads. Restore runs each task here.
class TaskPool {
public:
	TaskPool() = default;
	~TaskPool() = default;

	template <class F, class... Args>
	auto addTask(F &&f, Args &&...args) -> std::future<typename std::result_of<F(Args...)>::type> {
		using R = typename std::result_of<F(Args...)>::type;
		std::packaged_task<R()> task(std::bind(std::forward<F>(f), std::forward<Args>(args)...));
		std::future<R> res = task.get_future();
		task();
		return res;
	}

	size_t extraThreadCount() const { return 0; }
};

extern TaskPool globalTaskPool;

#endif

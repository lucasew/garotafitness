#include "task_pool.h"

extern "C" int pref_extra_threads(void)
	__attribute__((import_module("env"), import_name("pref_extra_threads")));

size_t TaskPool::extraThreadCount() const {
	int n = pref_extra_threads();
	if (n <= 0) {
		return 0;
	}
	return (size_t)n;
}

TaskPool globalTaskPool;

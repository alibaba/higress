package utils

import (
	"errors"
)

// FixedQueue 实现了一个固定容量的环形缓冲区队列
// 当队列满时，新元素会覆盖最旧的元素
type FixedQueue[T any] struct {
	data []T
	head int
	tail int
	size int
	cap  int
}

// NewFixed 创建一个指定容量的固定队列
func NewFixedQueue[T any](capacity int) *FixedQueue[T] {
	if capacity <= 0 {
		capacity = 16
	}
	return &FixedQueue[T]{
		data: make([]T, capacity),
		head: 0,
		tail: 0,
		size: 0,
		cap:  capacity,
	}
}

// Enqueue 入队操作
// 如果队列已满，会覆盖最旧的元素
func (q *FixedQueue[T]) Enqueue(item T) {
	if q.size < q.cap {
		// 队列未满，正常插入
		q.data[q.tail] = item
		q.tail = (q.tail + 1) % q.cap
		q.size++
	} else {
		// 队列已满，覆盖最旧元素
		q.data[q.tail] = item
		q.head = (q.head + 1) % q.cap // 移动head，丢弃最旧元素
		q.tail = (q.tail + 1) % q.cap // tail正常移动
		// size保持不变（仍然是cap）
	}
}

// Dequeue 出队操作
func (q *FixedQueue[T]) Dequeue() (T, error) {
	var zero T
	if q.size == 0 {
		return zero, errors.New("queue is empty")
	}

	item := q.data[q.head]
	// 清除引用，避免内存泄漏
	var zeroVal T
	q.data[q.head] = zeroVal

	q.head = (q.head + 1) % q.cap
	q.size--

	return item, nil
}

// Peek 查看队头元素但不移除
func (q *FixedQueue[T]) Peek() (T, error) {
	var zero T
	if q.size == 0 {
		return zero, errors.New("queue is empty")
	}
	return q.data[q.head], nil
}

// Size 返回队列中元素的数量
func (q *FixedQueue[T]) Size() int {
	return q.size
}

// Capacity 返回队列的固定容量
func (q *FixedQueue[T]) Capacity() int {
	return q.cap
}

// IsEmpty 判断队列是否为空
func (q *FixedQueue[T]) IsEmpty() bool {
	return q.size == 0
}

// IsFull 判断队列是否已满
func (q *FixedQueue[T]) IsFull() bool {
	return q.size == q.cap
}

// OverwriteCount 返回被覆盖的元素数量
// 注意：这个实现中我们不直接跟踪覆盖次数，
// 但可以通过其他方式计算（如果需要的话）
func (q *FixedQueue[T]) OverwriteCount() int {
	// 如果需要跟踪覆盖次数，可以添加一个字段
	// 目前这个实现不提供此功能
	return 0
}

// Clear 清空队列
func (q *FixedQueue[T]) Clear() {
	// 清除所有引用
	for i := 0; i < q.size; i++ {
		idx := (q.head + i) % q.cap
		var zero T
		q.data[idx] = zero
	}
	q.head = 0
	q.tail = 0
	q.size = 0
}

// ToSlice 返回队列元素的切片副本（按队列顺序，从最旧到最新）
func (q *FixedQueue[T]) ToSlice() []T {
	if q.size == 0 {
		return []T{}
	}

	result := make([]T, q.size)
	if q.head <= q.tail || q.size == q.cap {
		if q.head < q.tail {
			// 数据连续且未满
			copy(result, q.data[q.head:q.tail])
		} else {
			// 数据连续但已满（head == tail）
			// 或者数据跨越边界
			if q.head == q.tail && q.size == q.cap {
				// 已满且head == tail的情况
				copy(result, q.data[q.head:])
				if len(result) > q.cap-q.head {
					copy(result[q.cap-q.head:], q.data[:q.tail])
				}
			} else {
				// 跨越边界
				copy(result, q.data[q.head:])
				copy(result[q.cap-q.head:], q.data[:q.tail])
			}
		}
	} else {
		// 跨越边界的情况
		copy(result, q.data[q.head:])
		copy(result[q.cap-q.head:], q.data[:q.tail])
	}

	return result
}

// Oldest 返回最旧的元素（队头）
func (q *FixedQueue[T]) Oldest() (T, error) {
	return q.Peek()
}

// Newest 返回最新的元素（队尾的前一个元素）
func (q *FixedQueue[T]) Newest() (T, error) {
	var zero T
	if q.size == 0 {
		return zero, errors.New("queue is empty")
	}

	// tail指向下一个插入位置，所以最新元素在 (tail - 1 + cap) % cap
	newestIndex := (q.tail - 1 + q.cap) % q.cap
	return q.data[newestIndex], nil
}

// ForEach 对队列中的每个元素执行回调函数
func (q *FixedQueue[T]) ForEach(fn func(index int, item T)) {
	for i := 0; i < q.size; i++ {
		idx := (q.head + i) % q.cap
		fn(i, q.data[idx])
	}
}

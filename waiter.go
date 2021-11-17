package main

import (
	"sync"
)

// concurrentWaiter works as sync.WaitGroup, but it allows to use `Add` and `Wait` concurrently.
//
// sync.WaitGroup doesn't allow for that, see those links for detail:
// - https://github.com/golang/go/issues/23842
// - https://cs.opensource.google/go/go/+/refs/tags/go1.16.7:src/sync/waitgroup.go;l=88
type concurrentWaiter struct {
	state int

	// condZero has 2 functions here:
	// - it is a mutex for the state variable,
	// - it is a condition for `state == 0`, that allows waking up waiting goroutines.
	condZero *sync.Cond

	// allowNegative is used only for testing.
	// It allows calling `Done` and `Add` randomly, without taking care about the order.
	allowNegative bool
}

func newConcurrentWaiter() *concurrentWaiter {
	return &concurrentWaiter{
		condZero: sync.NewCond(new(sync.Mutex)),
	}
}

func (w *concurrentWaiter) Add(n uint) {
	w.condZero.L.Lock()
	defer w.condZero.L.Unlock()

	w.state += int(n)
}

func (w *concurrentWaiter) Done() {
	w.condZero.L.Lock()
	defer w.condZero.L.Unlock()

	w.state--
	if w.state < 0 && !w.allowNegative {
		panic("state is negative")
	}
	if w.state == 0 {
		w.condZero.Broadcast()
	}
}

func (w *concurrentWaiter) Wait() {
	w.condZero.L.Lock()
	defer w.condZero.L.Unlock()

	if w.state != 0 {
		w.condZero.Wait()
	}
}

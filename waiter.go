package syncwg

import (
	"sync"
)

// WaitGroup works similarly to sync.WaitGroup, but allows to use `Add` and `Wait` concurrently.
//
// sync.WaitGroup doesn't allow for that. See those links for detail:
// - https://github.com/golang/go/issues/23842
// - https://cs.opensource.google/go/go/+/refs/tags/go1.16.7:src/sync/waitgroup.go;l=88
//
//
// A WaitGroup waits for a collection of jobs to finish.
// Every time there is a "job" to do, Add is called to set the number of
// jobs to wait for. Add will return true if the job is allowed, or false otherwise.
// Then each of the jobs runs and calls Done when finished. At the same time,
// Wait can be used to block until all jobs have finished.
// The WaitGroup can be marked as "finished" by calling Finish.
// After the group is finished, no more jobs are accepted. Subsequent calls to Add will return false.
type WaitGroup struct {
	counter int
	done    bool

	// condZero has 2 functions here:
	// - it is a mutex for the state variable,
	// - it is a condition for `state == 0`, that allows waking up waiting goroutines.
	condZero *sync.Cond
}

// New creates a new WaitGroup.
func New() *WaitGroup {
	return &WaitGroup{
		condZero: sync.NewCond(new(sync.Mutex)),
	}
}

// Add adds delta, which may be negative, to the WaitGroup counter.
// If the counter becomes zero, all goroutines blocked on Wait are released.
// If the counter goes negative, Add panics.
//
// Note that calls with a positive delta that occur when the counter is zero
// must happen before a Wait. Calls with a negative delta, or calls with a
// positive delta that start when the counter is greater than zero, may happen
// at any time.
// Typically this means the calls to Add should execute before the statement
// creating the job or other event to be waited for.
// If a WaitGroup is reused to wait for several independent sets of events,
// new Add calls must happen after all previous Wait calls have returned.
func (w *WaitGroup) Add(n int) bool {
	w.condZero.L.Lock()
	defer w.condZero.L.Unlock()

	if w.done {
		// Not allowed to add any more work.
		return false
	}

	w.counter += n
	if w.counter < 0 {
		panic("syncwg: negative WaitGroup counter")
	}
	if w.counter == 0 {
		w.condZero.Broadcast()
	}
	return true
}

// Done decrements the WaitGroup counter by one.
func (w *WaitGroup) Done() {
	w.condZero.L.Lock()
	defer w.condZero.L.Unlock()

	w.counter--
	if w.counter < 0 {
		panic("syncwg: negative WaitGroup counter")
	}
	if w.counter == 0 {
		w.condZero.Broadcast()
	}
}

// Wait blocks until the WaitGroup counter is zero.
func (w *WaitGroup) Wait() {
	w.condZero.L.Lock()
	defer w.condZero.L.Unlock()

	for w.counter != 0 {
		w.condZero.Wait()
	}
}

// Finish makes group not accepting any more work.
// Subsequent calls to Add() will return false.
func (w *WaitGroup) Finish() {
	w.condZero.L.Lock()
	defer w.condZero.L.Unlock()

	w.done = true
}

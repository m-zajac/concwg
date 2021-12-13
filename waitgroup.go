package concwg

type state struct {
	counter  int
	finished bool
	waiters  []chan struct{}
}

func (s *state) notifyWaiters() {
	if s.counter != 0 {
		return
	}

	for _, c := range s.waiters {
		close(c)
	}
	s.waiters = nil
}

// WaitGroup works similarly to sync.WaitGroup, but allows to use `Add` and `Wait` concurrently.
//
// sync.WaitGroup doesn't allow for that. See those links for detail:
// - https://github.com/golang/go/issues/23842
// - https://cs.opensource.google/go/go/+/refs/tags/go1.16.7:src/sync/waitgroup.go;l=88
//
//
// A WaitGroup waits for a collection of jobs to finish.
// Every time there is a "job" to do, Add is called to set the number of jobs to wait for.
// Then each of the jobs runs and calls Done when finished. At the same time,
// Wait can be used to block until all jobs have finished.
type WaitGroup struct {
	s chan *state
}

// New creates a new WaitGroup.
func New() *WaitGroup {
	s := make(chan *state, 1)
	s <- &state{}
	return &WaitGroup{
		s: s,
	}
}

// Add adds delta, which may be negative, to the WaitGroup counter.
// If the counter becomes zero, all goroutines blocked on Wait are released.
// If the counter goes negative, Add panics.
//
// Add returns true if the counter was incremented and it is safe to perform the job.
// If it returns true, it means that the group was marked as "finished" and won't accept any more jobs.
//
// Note that calls with a positive delta that occur when the counter is zero
// must happen before a Wait. Calls with a negative delta, or calls with a
// positive delta that start when the counter is greater than zero, may happen
// at any time.
// Typically this means the calls to Add should execute before the statement
// creating the job or other event to be waited for.
//
// WaitGroup is not designed to be reused. After call to Finish it will never accept any new jobs.
func (w *WaitGroup) Add(n int) bool {
	s := <-w.s
	defer func() { w.s <- s }()

	if s.finished {
		return false
	}

	s.counter += n
	if s.counter < 0 {
		panic("concwg: negative WaitGroup counter")
	}

	s.notifyWaiters()

	return true
}

// Done decrements the WaitGroup counter by one.
func (w *WaitGroup) Done() {
	s := <-w.s
	defer func() { w.s <- s }()

	s.counter--
	if s.counter < 0 {
		panic("concwg: negative WaitGroup counter")
	}

	s.notifyWaiters()
}

// Wait blocks until the WaitGroup counter is zero.
func (w *WaitGroup) Wait() {
	s := <-w.s

	s.finished = true

	if s.counter == 0 {
		w.s <- s
		return
	}

	wait := make(chan struct{})
	s.waiters = append(s.waiters, wait)
	w.s <- s

	<-wait
}

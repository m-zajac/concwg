//nolint:gosec // We're using plain random for simplicity.
package syncwg

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

//nolint:funlen // Long test.
func Test_concurrentWaiter(t *testing.T) {
	t.Parallel()

	type op struct {
		numAdds    int
		numDones   int
		shouldWait bool
	}

	tests := []struct {
		name string
		ops  []op
	}{
		{
			name: "0/0",
			ops: []op{
				{
					numAdds:    0,
					numDones:   0,
					shouldWait: false,
				},
			},
		},
		{
			name: "1/0",
			ops: []op{
				{
					numAdds:    1,
					numDones:   0,
					shouldWait: true,
				},
			},
		},
		{
			name: "50/50",
			ops: []op{
				{
					numAdds:    50,
					numDones:   50,
					shouldWait: false,
				},
			},
		},
		{
			name: "51/50",
			ops: []op{
				{
					numAdds:    51,
					numDones:   50,
					shouldWait: true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			w := New()

			for i, op := range tt.ops {
				t.Run(fmt.Sprintf("op-%d", i+1), func(t *testing.T) {
					var wg sync.WaitGroup
					for i := 0; i < op.numAdds; i++ {
						wg.Add(1)
						go func() {
							defer wg.Done()

							d := float64(time.Millisecond*100) * rand.Float64()
							time.Sleep(time.Duration(d))

							w.Add(1)
						}()
					}
					for i := 0; i < op.numDones; i++ {
						wg.Add(1)
						go func() {
							defer wg.Done()

							d := float64(time.Millisecond*100) * rand.Float64()
							time.Sleep(time.Duration(d))

							w.Done()
						}()
					}
					wg.Wait()

					done := waiterWait(w)

					select {
					case <-done:
						if op.shouldWait {
							t.Error("waiter should wait, but didn't")
						}
					case <-time.After(time.Second):
						if !op.shouldWait {
							t.Error("waiter shouldn't wait, but did")
						}
					}
				})
			}
		})
	}
}

// waiterWait spins 10 goroutines that `Wait`, and returns a chan that is closed, when all waits return.
func waiterWait(w *WaitGroup) <-chan struct{} {
	var chs []chan struct{}
	for i := 0; i < 10; i++ {
		ch := make(chan struct{})
		chs = append(chs, ch)
		go func() {
			w.Wait()
			close(ch)
		}()
	}

	done := make(chan struct{})
	go func() {
		for _, ch := range chs {
			<-ch
		}
		close(done)
	}()
	return done
}

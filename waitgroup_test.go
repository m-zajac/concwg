//nolint:gosec // We're using plain random for simplicity.
package concwg_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/m-zajac/concwg"
	"github.com/stretchr/testify/assert"
)

//nolint:funlen // Long test.
func TestWaiter(t *testing.T) {
	t.Parallel()

	type op struct {
		numAdds    int
		numDones   int
		shouldWait bool
	}

	tests := map[string]struct {
		ops []op
	}{
		"0/0": {
			ops: []op{
				{
					numAdds:    0,
					numDones:   0,
					shouldWait: false,
				},
			},
		},
		"1/0": {
			ops: []op{
				{
					numAdds:    1,
					numDones:   0,
					shouldWait: true,
				},
			},
		},
		"50/50": {
			ops: []op{
				{
					numAdds:    50,
					numDones:   50,
					shouldWait: false,
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			wg := concwg.New()

			for i, op := range tt.ops {
				op := op
				t.Run(fmt.Sprintf("op-%d", i+1), func(t *testing.T) {
					var swg sync.WaitGroup
					adds := make(chan struct{}, op.numAdds)
					for i := 0; i < op.numAdds; i++ {
						swg.Add(1)
						go func() {
							defer func() {
								if err := recover(); err != nil {
									panic(fmt.Errorf("op: %#v, err: %v", op, err))
								}
							}()
							defer swg.Done()

							d := float64(time.Millisecond*100) * rand.Float64()
							time.Sleep(time.Duration(d))

							if ok := wg.Add(1); !ok {
								panic("wg.Add returned false")
							}
							adds <- struct{}{}
						}()
					}
					for i := 0; i < op.numDones; i++ {
						swg.Add(1)
						go func() {
							defer func() {
								if err := recover(); err != nil {
									panic(fmt.Errorf("op: %#v, err: %v", op, err))
								}
							}()
							defer swg.Done()

							d := float64(time.Millisecond*100) * rand.Float64()
							time.Sleep(time.Duration(d))

							<-adds
							wg.Done()
						}()
					}
					swg.Wait()

					// After finishing waitgroup, it should not accept any more jobs.
					wg.Finish()
					ok := wg.Add(1)
					assert.False(t, ok)

					done := waiterWait(wg)

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

func TestWaiterTorture(t *testing.T) {
	for i := 0; i < 100; i++ {
		jobs := rand.Intn(1000)
		adds := make(chan struct{}, jobs)
		dones := make(chan struct{}, jobs)
		concurrentWaits := make(chan struct{}, jobs)

		wg := concwg.New()
		for j := 0; j < jobs; j++ {
			added := make(chan struct{})
			go func() {
				ok := wg.Add(1)
				if !ok {
					panic("wg.Add returned false")
				}
				adds <- struct{}{}
				close(added)
			}()
			go func() {
				<-added
				time.Sleep(time.Duration(rand.Intn(100000))) // Up to 0.1 millisecond.
				wg.Done()
				dones <- struct{}{}
			}()
			go func() {
				wg.Wait()
				concurrentWaits <- struct{}{}
			}()
		}

		// After all the jobs are added mark wg as finished.
		for j := 0; j < jobs; j++ {
			<-adds
		}
		wg.Finish()
		for i := 0; i < 10; i++ {
			go func() {
				if ok := wg.Add(1); ok {
					panic("wg.Add returned true after finishing")
				}
			}()
		}

		<-waiterWait(wg)

		for j := 0; j < jobs; j++ {
			select {
			case <-dones:
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("timeout")
			}

			select {
			case <-concurrentWaits:
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("timeout")
			}
		}
	}
}

// waiterWait spins 10 goroutines that `Wait`, and returns a chan that is closed, when all waits return.
func waiterWait(wg *concwg.WaitGroup) <-chan struct{} {
	var chs []chan struct{}
	for i := 0; i < 10; i++ {
		ch := make(chan struct{})
		chs = append(chs, ch)
		go func() {
			wg.Wait()
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

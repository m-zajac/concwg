//nolint:gosec // We're using plain random for simplicity.
package concwg_test

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/m-zajac/concwg"
)

//nolint:funlen // Long test.
func TestWaiter(t *testing.T) {
	t.Parallel()

	type op struct {
		numAdds     int
		numDones    int
		shouldWait  bool
		shouldPanic bool
	}

	tests := map[string]struct {
		ops []op
	}{
		"0/0": {
			ops: []op{
				{
					numAdds:     0,
					numDones:    0,
					shouldWait:  false,
					shouldPanic: false,
				},
			},
		},
		"1/0": {
			ops: []op{
				{
					numAdds:     1,
					numDones:    0,
					shouldWait:  true,
					shouldPanic: false,
				},
			},
		},
		"50/50": {
			ops: []op{
				{
					numAdds:     50,
					numDones:    50,
					shouldWait:  false,
					shouldPanic: false,
				},
			},
		},
		"51/50": {
			ops: []op{
				{
					numAdds:     51,
					numDones:    50,
					shouldWait:  true,
					shouldPanic: true,
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			wg := concwg.New()

			for i, op := range tt.ops {
				t.Run(fmt.Sprintf("op-%d", i+1), func(t *testing.T) {
					panics := make(chan struct{}, 1)
					var swg sync.WaitGroup
					for i := 0; i < op.numAdds; i++ {
						swg.Add(1)
						go func() {
							defer func() {
								if err := recover(); err != nil {
									panics <- struct{}{}
								}
							}()
							defer swg.Done()

							d := float64(time.Millisecond*100) * rand.Float64()
							time.Sleep(time.Duration(d))

							wg.Add(1)
						}()
					}
					for i := 0; i < op.numDones; i++ {
						swg.Add(1)
						go func() {
							defer func() {
								if err := recover(); err != nil {
									panics <- struct{}{}
								}
							}()
							defer swg.Done()

							d := float64(time.Millisecond*100) * rand.Float64()
							time.Sleep(time.Duration(d))

							wg.Done()
						}()
					}
					swg.Wait()

					done := waiterWait(wg)

					select {
					case <-panics:
						if !op.shouldPanic {
							t.Error("unexpected panic")
						}
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
	for i := 0; i < 10; i++ {
		jobs := rand.Intn(10000)
		dones := make(chan struct{}, jobs)
		concurrentWaits := make(chan struct{}, jobs)

		wg := concwg.New()
		for j := 0; j < jobs; j++ {
			go func() {
				wg.Add(1)

				go func() {
					time.Sleep(time.Duration(rand.Intn(1000000))) // Up to 1 millisecond.
					wg.Done()
					dones <- struct{}{}
				}()
			}()
			go func() {
				<-waiterWait(wg)
				concurrentWaits <- struct{}{}
			}()
		}
		wg.Wait()

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

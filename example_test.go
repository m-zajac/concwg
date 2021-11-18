package syncwg_test

import (
	"net/http"
	"syncwg"
)

// This example fetches several URLs concurrently,
// using a WaitGroup to block until all the fetches are complete.
func ExampleWaitGroup() {
	wg := syncwg.New()
	fetchURLs := func() {
		var urls = []string{
			"http://www.golang.org/",
			"http://www.google.com/",
			"http://www.somestupidname.com/",
		}
		for _, url := range urls {
			// Increment the WaitGroup counter.
			wg.Add(1)
			// Launch a goroutine to fetch the URL.
			go func(url string) {
				// Decrement the counter when the goroutine completes.
				defer wg.Done()
				// Fetch the URL.
				http.Get(url)
			}(url)
		}
	}

	fetchURLs()

	// Wait for all HTTP fetches to complete.
	wg.Wait()
}

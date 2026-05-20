package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	fmt.Println("Starting load test...")
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Simulate a call
			resp, err := http.Get("http://localhost:8080/healthz")
			if err != nil {
				fmt.Printf("Agent %d failed: %v\n", id, err)
				return
			}
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
	fmt.Println("Load test complete.")
}

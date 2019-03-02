/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/knative/serving/test"
)

// Algorithm from https://stackoverflow.com/a/21854246

// Only primes less than or equal to N will be generated
func primes(N int) []int {

	var x, y, n int
	nsqrt := math.Sqrt(float64(N))

	isPrime := make([]bool, N)

	for x = 1; float64(x) <= nsqrt; x++ {
		for y = 1; float64(y) <= nsqrt; y++ {
			n = 4*(x*x) + y*y
			if n <= N && (n%12 == 1 || n%12 == 5) {
				isPrime[n] = !isPrime[n]
			}
			n = 3*(x*x) + y*y
			if n <= N && n%12 == 7 {
				isPrime[n] = !isPrime[n]
			}
			n = 3*(x*x) - y*y
			if x > y && n <= N && n%12 == 11 {
				isPrime[n] = !isPrime[n]
			}
		}
	}

	for n = 5; float64(n) <= nsqrt; n++ {
		if isPrime[n] {
			for y = n * n; y < N; y += n * n {
				isPrime[y] = false
			}
		}
	}

	isPrime[2] = true
	isPrime[3] = true

	primes := make([]int, 0, 1270606)
	for x = 0; x < len(isPrime)-1; x++ {
		if isPrime[x] {
			primes = append(primes, x)
		}
	}

	// primes is now a slice that contains all primes numbers up to N
	return primes
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		p := primes(400000)
		largest := p[len(p)-1]
		msg := fmt.Sprintf("The largest prime under 400000 is %d. Enjoy your noodles!", largest)
		fmt.Fprintf(w, msg)
		log.Printf(msg)
	}()
	go func() {
		defer wg.Done()
		start := time.Now()
		time.Sleep(time.Second)
		msg := fmt.Sprintf("Slept for %v.", time.Since(start))
		fmt.Fprintf(w, msg)
		log.Printf(msg)
	}()
	wg.Wait()
}

func main() {
	test.ListenAndServeGracefully(":8080", handler)
}

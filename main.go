package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/eaciit/toolkit"
)

var (
	host      = flag.String("h", "", "host")
	testCount = flag.Int("n", 100, "number of test")
	period    = flag.Int("t", 60, "second to test")
)

type callStat struct {
	sync.Mutex
	Count         int
	TotalDuration time.Duration
	AvgDuration   time.Duration
}

func (c *callStat) addResult(d time.Duration) {
	c.Lock()
	c.Count++
	c.TotalDuration += d
	c.AvgDuration = c.TotalDuration / time.Duration(c.Count)
	c.Unlock()
}

func main() {
	flag.Parse()
	fmt.Printf("Running stress test on %s for %v with %d test per iteration\n", *host, time.Duration(*period)*time.Second, *testCount)

	if *host == "" {
		fmt.Println("host is not defined")
		return
	}

	t0 := time.Now()
	for {
		if time.Since(t0) < time.Duration(*period)*time.Second {

			ress := map[string]*callStat{
				"OK":   new(callStat),
				"Fail": new(callStat),
			}

			mtx := new(sync.Mutex)
			reasons := map[string]int{}

			wg := new(sync.WaitGroup)
			wg.Add(*testCount)

			for i := 0; i < *testCount; i++ {
				go func() {
					ok := true

					defer func() {
						if ok {
							fmt.Printf(".")
						} else {
							fmt.Printf("x")
						}
						wg.Done()
					}()

					var diff time.Duration
					call0 := time.Now()
					resp, err := toolkit.HttpCall(*host, http.MethodGet, []byte{}, nil)
					if err != nil || resp.StatusCode >= 400 {
						diff = time.Since(call0)
						stat := ress["Fail"]
						stat.addResult(diff)
						ok = false

						mtx.Lock()
						if err != nil {
							reasons[err.Error()]++
						} else {
							reasons[resp.Status]++
						}
						mtx.Unlock()
						return
					}

					diff = time.Since(call0)
					stat := ress["OK"]
					stat.addResult(diff)
				}()
			}

			wg.Wait()
			fmt.Println()
			if ress["OK"].Count > 0 {
				fmt.Printf("OK: %d Average: %v\n", ress["OK"].Count, ress["OK"].AvgDuration)
			}

			if ress["Fail"].Count > 0 {
				fmt.Printf("Fail: %d Average: %v\n", ress["Fail"].Count, ress["Fail"].AvgDuration)
				for k, v := range reasons {
					fmt.Printf("[%d = %.1f%%] %s\n", v, float64(v*100)/float64(*testCount), k)
				}
			}
		} else {
			break
		}
	}
}

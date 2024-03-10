package main

import (
	"fmt"
	"net/http"
	"sync"
)

func main() {
	wg := sync.WaitGroup{}
	for x := 0; x < 4; x++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				fmt.Println(http.Get("http://dev.cecilialiu.cc/healthz"))
			}
		}()
	}
	wg.Wait()

}

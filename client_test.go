package gorews

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestNewGorewsClient(t *testing.T) {
	client := NewGorewsClient()
	var headers http.Header
	err := client.Start("ws://demos.kaazing.com/echo", headers, time.Second, time.Second, time.Second)
	if err != nil {
		fmt.Println(err)
	}

	c1 := sync.WaitGroup{}
	c2 := sync.WaitGroup{}
	c1.Add(1)
	c2.Add(1)

	go func() {

		for i := 0; i < 5; i++ {
			time.Sleep(time.Second)
			client.Outgoing <- []byte(time.Now().String())
		}
		c1.Done()
	}()

	go func() {
		for {
			select {
			case m, ok := <-client.Incoming:
				if !ok {
					c2.Done()
					return
				}
				fmt.Println(string(m))
			}
		}
	}()
	c1.Wait()
	client.Stop()
	c2.Wait()

}

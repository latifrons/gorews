package gorews

import (
	"fmt"
	"net/http"
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
	go func() {
		for {
			time.Sleep(time.Second)
			client.Outgoing <- []byte(time.Now().String())
		}
	}()

	time.Sleep(time.Second * 5)

	client.Stop()

}

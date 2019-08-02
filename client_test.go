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

	time.Sleep(time.Second * 20)

	client.Stop()

}

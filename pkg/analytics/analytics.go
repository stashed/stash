package analytics

import (
	"sync"

	"github.com/appscode/log"
	ga "github.com/jpillora/go-ogle-analytics"
)

const (
	id = "UA-62096468-15"
)

var (
	mu     sync.Mutex
	client *ga.Client
)

func mustNewClient() *ga.Client {
	client, err := ga.NewClient(id)
	if err != nil {
		log.Fatalln(err)
	}
	return client
}

func Enable() {
	mu.Lock()
	defer mu.Unlock()
	client = mustNewClient()
}

func Disable() {
	mu.Lock()
	defer mu.Unlock()
	client = nil
}

func send(e *ga.Event) {
	mu.Lock()
	c := client
	mu.Unlock()

	if c == nil {
		return
	}
	c.Send(e)
}

func SendEvent(category string, action, label string) {
	event := ga.NewEvent(category, action)
	if label != "" {
		event = event.Label(label)
	}
	send(event)
}

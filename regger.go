package main

import (
	"encoding/json"
	"flag"
	"log"
	"time"

	nats "github.com/nats-io/go-nats"
)

type Route struct {
	Host string      `json:"host"`
	Port int         `json:"port"`
	Uris []string    `json:"uris"`
	Tags interface{} `json:"tags"`
}

func main() {

	proxiedHostPtr := flag.String("proxied-host", "localhost", "host to proxy to")
	proxiedPortPtr := flag.Int("proxied-port", 8080, "port to proxy to")
	urlPtr := flag.String("url", "host1.fqdn.com", "url")
	natsAddrPtr := flag.String("nats-address", "nats://localhost:4222", "NATS address host:port")

	flag.Parse()

	nc, err := nats.Connect(*natsAddrPtr)

	if err != nil {
		log.Fatal("Can't connect to NATS. Error:", err)
	}

	defer nc.Close()

	emptyTags := make(map[string]string)

	route := Route{
		Host: *proxiedHostPtr,
		Port: *proxiedPortPtr,
		Uris: []string{*urlPtr},
		Tags: emptyTags,
	}

	// publish the route during start
	publish(nc, route)

	// publish route until 'quit' is issued every 'interval'
	ticker := time.NewTicker(30 * time.Second)
	quit := make(chan struct{})

	for {
		select {
		case <-ticker.C:
			publish(nc, route)
		case <-quit:
			ticker.Stop()
			return
		}
	}
}

func publish(nc *nats.Conn, route Route) {

	// '{"host":"127.0.0.1","port":4567,"uris":["my_first_url.vcap.me","my_second_url.vcap.me"],"tags":{"another_key":"another_value","some_key":"some_value"}}'

	if b, err := json.Marshal(route); err != nil {
		log.Println("Marshalling to JSON failed. Can't marshall", route)
	} else {

		subject := "router.register"
		msg := b

		nc.Publish("router.register", msg)
		timeoutErr := nc.FlushTimeout(5 * time.Second)

		if timeoutErr != nil {
			log.Printf("Timeout publishing [%s] : '%s'. Error: %s\n", subject, msg, timeoutErr)
		} else if err := nc.LastError(); err != nil {
			log.Printf("Error publishing [%s] : '%s'. Error: %s\n", subject, msg, err)
		} else {
			log.Printf("Published [%s] : '%s'\n", subject, msg)
		}
	}

}

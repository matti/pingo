package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sync"

	probing "github.com/prometheus-community/pro-bing"
)

type Reply struct {
	host string
	ms   int64
}

var replies = make(chan Reply, 1)

func ping(ctx context.Context, host string) {
	pinger, err := probing.NewPinger(host)
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}

	pinger.Interval = time.Second
	// wtf
	pinger.Timeout = time.Second * 100000

	pinger.Count = -1

	pinger.Size = 24
	pinger.TTL = 64

	pinger.SetPrivileged(false)

	pinger.OnRecv = func(pkt *probing.Packet) {
		replies <- Reply{
			host: host,
			ms:   pkt.Rtt.Milliseconds(),
		}
	}

	pinger.OnDuplicateRecv = func(pkt *probing.Packet) {
		fmt.Printf("%-7d", pkt.Rtt.Milliseconds())
		fmt.Println("DUP")
	}

	pinger.OnFinish = func(stats *probing.Statistics) {
		fmt.Println("finish", stats)
	}

	go func() {
		if err := pinger.Run(); err != nil {
			panic(err)
		}

		fmt.Println("pinger run stop")
	}()

	<-ctx.Done()
	fmt.Println("ctx done")
	pinger.Stop()
}

func printer(hosts []string) {
	for {
		gots := make(map[string]Reply)

		for {
			r := <-replies
			gots[r.host] = r

			if len(gots) == len(hosts) {
				break
			}
		}

		var previous int
		for index, host := range hosts {
			size := int(gots[host].ms)
			if index == 0 {
				previous = size
			} else {
				size = size - previous
			}

			for i := 0; i < size; i++ {
				if index == 0 {
					fmt.Print("|")
				} else {
					fmt.Print(".")
				}
			}
		}
		fmt.Println("")
	}
}
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigs

		cancel()
	}()

	flag.Parse()

	go printer(flag.Args())

	var wg sync.WaitGroup
	for _, host := range flag.Args() {
		wg.Add(1)
		go func(h string) {
			ping(ctx, h)
			wg.Done()
		}(host)
	}

	wg.Wait()
}

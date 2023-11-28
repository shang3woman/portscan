package main

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	var hosts string
	var ports string
	var count int
	flag.StringVar(&hosts, "h", "", "hosts")
	flag.StringVar(&ports, "p", "", "ports")
	flag.IntVar(&count, "c", 1000, "")
	flag.Parse()
	if len(hosts) == 0 {
		flag.Usage()
		return
	}
	if count <= 0 {
		count = 1000
	}
	ch := make(chan string)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go thread(&wg, ch)
	}
	hostarr := strings.Split(hosts, ",")
	var portarr []string
	if len(ports) == 0 {
		for i := 1; i < 65536; i++ {
			portarr = append(portarr, strconv.Itoa(i))
		}
	} else {
		portarr = strings.Split(ports, ",")
	}
	for i := 0; i < len(hostarr); i++ {
		for j := 0; j < len(portarr); j++ {
			ch <- fmt.Sprintf("%s:%s", hostarr[i], portarr[j])
		}
	}
	close(ch)
	wg.Wait()
}

func thread(wg *sync.WaitGroup, ch chan string) {
	defer wg.Done()
	for address := range ch {
		if connect(address) {
			fmt.Println(address)
		}
	}
}

func connect(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 4*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

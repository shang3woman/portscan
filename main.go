package main

import (
	"bytes"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Hosts(cidr string) []string {
	if len(cidr) == 0 {
		return []string{}
	}
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return []string{cidr}
	}
	prefix, _ := netip.ParsePrefix(ipnet.String())
	var ips []string
	for addr := prefix.Addr(); prefix.Contains(addr); addr = addr.Next() {
		ips = append(ips, addr.String())
	}
	return ips
}

func extractHosts(str string) []string {
	tmparr := strings.Split(str, ",")
	var results []string
	for _, v := range tmparr {
		results = append(results, Hosts(v)...)
	}
	return results
}

func main() {
	var hosts string
	var ports string
	var count int
	flag.StringVar(&hosts, "h", "", "hosts")
	flag.StringVar(&ports, "p", "", "ports")
	flag.IntVar(&count, "c", 1000, "")
	flag.Parse()
	hostsarr := extractHosts(hosts)
	if len(hostsarr) == 0 {
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
	var portarr []string
	if len(ports) == 0 {
		for i := 1; i < 65536; i++ {
			portarr = append(portarr, strconv.Itoa(i))
		}
	} else {
		portarr = strings.Split(ports, ",")
	}
	for i := 0; i < len(hostsarr); i++ {
		for j := 0; j < len(portarr); j++ {
			ch <- fmt.Sprintf("%s:%s", hostsarr[i], portarr[j])
		}
	}
	close(ch)
	wg.Wait()
}

func isChar(b byte) bool {
	if b >= 32 && b <= 126 {
		return true
	}
	return false
}

func hexBanner(banner []byte) string {
	var results bytes.Buffer
	for _, v := range banner {
		if isChar(v) {
			results.WriteByte(v)
		} else {
			results.WriteString(fmt.Sprintf("\\x%02x", v))
		}
	}
	return results.String()
}

func thread(wg *sync.WaitGroup, ch chan string) {
	defer wg.Done()
	for address := range ch {
		ok, banner := connect(address)
		if !ok {
			continue
		}
		if len(banner) != 0 {
			fmt.Printf("%s %s\n", address, hexBanner(banner))
		} else {
			fmt.Println(address)
		}
	}
}

func connect(address string) (bool, []byte) {
	conn, err := net.DialTimeout("tcp", address, 4*time.Second)
	if err != nil {
		return false, nil
	}
	req := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nAccept: */*\r\n\r\n", address)
	conn.Write([]byte(req))
	time.Sleep(2 * time.Second)
	conn.SetReadDeadline(time.Now().Add(4 * time.Second))
	var buffer [128]byte
	n, _ := conn.Read(buffer[:])
	conn.Close()
	return true, buffer[:n]
}

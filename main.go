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
		ok, httpBanner, rpcBanner := connect(address)
		if !ok {
			continue
		}
		var output string
		if len(httpBanner) != 0 {
			output = fmt.Sprintf("%s http %s", address, hexBanner(httpBanner))
		}
		if len(rpcBanner) != 0 {
			tmp := fmt.Sprintf("%s rpc %s", address, hexBanner(rpcBanner))
			if len(output) == 0 {
				output = tmp
			} else {
				output += "\n" + tmp
			}
		}
		if len(output) == 0 {
			fmt.Println(address)
		} else {
			fmt.Println(output)
		}
	}
}

var msrpc = []byte{0x5, 0x0, 0xb, 0x3, 0x10, 0x0, 0x0, 0x0, 0x78, 0x0, 0x28, 0x0, 0x3, 0x0, 0x0, 0x0, 0xb8, 0x10, 0xb8, 0x10, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0xa0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xc0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x46, 0x0, 0x0, 0x0, 0x0, 0x4, 0x5d, 0x88, 0x8a, 0xeb, 0x1c, 0xc9, 0x11, 0x9f, 0xe8, 0x8, 0x0, 0x2b, 0x10, 0x48, 0x60, 0x2, 0x0, 0x0, 0x0, 0xa, 0x2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4e, 0x54, 0x4c, 0x4d, 0x53, 0x53, 0x50, 0x0, 0x1, 0x0, 0x0, 0x0, 0x7, 0x82, 0x8, 0xa2, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x6, 0x1, 0xb1, 0x1d, 0x0, 0x0, 0x0, 0xf}

func connect(address string) (bool, []byte, []byte) {
	conn, err := net.DialTimeout("tcp", address, 4*time.Second)
	if err != nil {
		return false, nil, nil
	}
	httpReq := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nAccept: */*\r\n\r\n", address)
	httpRsp := readBanner(conn, []byte(httpReq), 4*time.Second)
	if len(httpRsp) != 0 {
		if index := bytes.Index(httpRsp, []byte("\r\n\r\n")); index != -1 {
			httpRsp = httpRsp[0:index]
		}
	}
	conn.Close()
	conn, err = net.DialTimeout("tcp", address, 4*time.Second)
	if err != nil {
		return true, httpRsp, nil
	}
	rpcRsp := readBanner(conn, []byte(msrpc), 4*time.Second)
	conn.Close()
	return true, httpRsp, rpcRsp
}

func readBanner(conn net.Conn, req []byte, timeout time.Duration) []byte {
	conn.Write([]byte(req))
	conn.SetReadDeadline(time.Now().Add(timeout))
	var ret []byte
	for {
		var tmp [1024]byte
		n, err := conn.Read(tmp[:])
		if err != nil {
			break
		}
		ret = append(ret, tmp[0:n]...)
	}
	return ret
}

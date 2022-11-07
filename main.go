package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// table entry struct
type addr_info struct {
	timestamp   int64
	num         int
	blocklisted bool
}

// configuration
var server_port string = "6410"
var adversarial bool = false
var public_ip bool = false

// global variables
var addr_map map[string]*addr_info
var mutex = &sync.RWMutex{}
var self_ip string

func main() {
	for i, arg := range os.Args[1:] {
		switch i {
		case 0:
			server_port = arg
		default:
			panic("too many arguments")
		}
	}

	addr_map = make(map[string]*addr_info) //init info list

	if public_ip {
		var err error
		self_ip, err = Http_get("https://api.ipify.org") //get self ip
		if err != nil {
			panic(err)
		}
	} else {

		addrs, err := net.InterfaceAddrs()
		if err != nil {
			panic(err)
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok &&
				!ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil && ipnet.IP.String()[:3] == "10." {
				self_ip = ipnet.IP.String()
				break
			}
		}
	}

	self_ip += ":" + server_port

	new_info := addr_info{time.Now().Unix(), 0, false} //create this computer's own record; default number to 0
	addr_map[self_ip] = &new_info                      //add to the address map

	go schedule()
	go server()

	loop()
}

func loop() {
	//start infinite command prompt
	var cmd string
	for {
		cmd = ""
		fmt.Print(">> ")
		fmt.Scanf("%s\n", &cmd)
		//determine what the cmd wants to do

		switch len(cmd) {
		case 0:
			continue
		case 1:
			char := int(cmd[0])

			if char == '?' {
				mutex.RLock()

				//print beautified addr_map
				for k, v := range addr_map {
					fmt.Println(k, " --> ", v.num)
				}

				mutex.RUnlock()
				continue
			} else if char >= '0' && char <= '9' {
				mutex.Lock()

				if entry, ok := addr_map[self_ip]; ok {
					(*entry).timestamp = time.Now().Unix()
					(*entry).num = char - '0'

					fmt.Println(self_ip, " --> ", char-'0')

					mutex.Unlock()
					continue
				}

				mutex.Unlock()
			} else if char == '!' {
				mutex.RLock()

				for k, v := range addr_map {
					fmt.Println(k + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(v.num) + "," + fmt.Sprint(v.blocklisted))
				}

				mutex.RUnlock()
				continue
			} else if char == 'a' {
				adversarial = !adversarial
				fmt.Println("Adversarial mode enabled:", adversarial)

				continue
			}
		default:
			var n1, n2, n3, n4 uint8
			var n5 uint16
			if n, err := fmt.Sscanf(cmd, "+%d.%d.%d.%d:%d\n", &n1, &n2, &n3, &n4, &n5); n == 5 && err == nil {
				go send_request(cmd[1:])
				continue
			}
		}
		fmt.Println("Invalid input")
	}
}

func update_addr_map(conn net.Conn) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	for count := 0; scanner.Scan() && count < 256; count++ {
		data := scanner.Text()

		var n1, n2, n3, n4 uint8
		var port uint16
		var t int64
		var d int
		//example: 128.84.213.13:5678,1630281124,1
		if n, err := fmt.Sscanf(data, "%d.%d.%d.%d:%d,%d,%d\n", &n1, &n2, &n3, &n4, &port, &t, &d); n == 7 && err == nil &&
			d >= 0 && d <= 9 && t <= time.Now().Unix() {

			ip := fmt.Sprintf("%d.%d.%d.%d", n1, n2, n3, n4)
			ip_address := net.ParseIP(ip)

			//check if private ip
			if ip_address == nil || (public_ip && ip_address.IsPrivate()) {
				break
			}

			//not going to check how many connections in the table are from this IP due to same wifi testing
			ip += ":" + fmt.Sprint(port)

			if entry, ok := addr_map[ip]; ok {
				ts := entry.timestamp
				if t > ts && !entry.blocklisted {
					(*entry).timestamp = t
					(*entry).num = d

					fmt.Printf("\n%s --> %d\n", ip, d)
				}
			} else {
				new_info := addr_info{t, d, false}
				addr_map[ip] = &new_info //golang feature: escape analysis

				fmt.Printf("\n%s --> %d\n", ip, d)
			}
		}
	}
}

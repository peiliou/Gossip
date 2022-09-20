package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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

// global variables
var addr_map map[string]*addr_info
var mutex = &sync.RWMutex{}
var self_ip string

func main() {
	addr_map = make(map[string]*addr_info) //init info list

	var err error
	self_ip, err = Http_get("https://api.ipify.org") //get self ip
	if err != nil {
		panic(err)
	}
	self_ip += ":" + server_port

	new_info := addr_info{time.Now().Unix(), 0, false} //create this computer's own record; default number to 0
	addr_map[self_ip] = &new_info                      //add to the address map

	go schedule()
	go server()

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
				//print beautified addr_map
				for k, v := range addr_map {
					fmt.Println(k, " --> ", v.num)
				}

				continue
			} else if char >= '0' && char <= '9' {
				if entry, ok := addr_map[self_ip]; ok {
					(*entry).timestamp = time.Now().Unix()
					(*entry).num = char - '0'

					fmt.Println(self_ip, " --> ", char-'0')

					continue
				}
			} else if char == '!' {
				for k, v := range addr_map {
					fmt.Println(k + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(v.num))
				}

				continue
			}
		default:
			var n1, n2, n3, n4 uint8
			var n5 uint16
			if n, err := fmt.Sscanf(cmd, "+%d.%d.%d.%d:%d\n", &n1, &n2, &n3, &n4, &n5); n == 5 && err == nil {
				new_info := addr_info{time.Now().Unix(), -1, false}

				mutex.Lock()

				ok := send_request(cmd[1:])
				if ok {
					addr_map[cmd[1:]] = &new_info //might overwrite existing record if duplicated
				}

				mutex.Unlock()

				continue
			}
		}
		fmt.Println("Invalid input")
	}
}

func send_request(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)

	if err != nil {
		fmt.Println(err.Error())
		if entry, ok := addr_map[addr]; ok {
			(*entry).blocklisted = true
		}
		return false
	} else {
		update_addr_map(conn)
		return true
	}
}

func update_addr_map(conn net.Conn) {
	defer conn.Close()

	for count := 0; count < 256; count++ {
		data, err := bufio.NewReader(conn).ReadString('\n')
		if len(data) == 0 && err != nil {
			break
		}

		var n1, n2, n3, n4 uint8
		var port uint16
		var t int64
		var d int
		//example: 128.84.213.13:5678,1630281124,1
		if n, err := fmt.Sscanf(data, "%d.%d.%d.%d:%d,%d\n", &n1, &n2, &n3, &n4, &port, &t, &d); n == 7 && err == nil &&
			d >= 0 && d <= 9 && t <= time.Now().Unix() {

			ip := fmt.Sprintf("%d.%d.%d.%d", n1, n2, n3, n4)
			ip_address := net.ParseIP(ip)

			//check if private ip
			if ip_address == nil || ip_address.IsPrivate() {
				break
			}

			ip += ":" + fmt.Sprint(port) //not going to check how many connections in the table are from this IP due to same wifi testing

			if entry, ok := addr_map[ip]; ok {
				ts := entry.timestamp
				if t > ts && !entry.blocklisted {
					(*entry).timestamp = t
					(*entry).num = d
				} else {
					(*entry).blocklisted = true
				}
			} else {
				new_info := addr_info{t, d, false}
				addr_map[ip] = &new_info //golang feature: escape analysis
			}

			fmt.Printf("\n%s --> %d\n", ip, d)
		}
	}
}

func Http_get(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func schedule() {
	for {
		time.Sleep(3000 * time.Millisecond)
		//select a random entry in its map ignoring its own entry  and try to set up a TCP/IP connection to a node for gossip
		mutex.Lock()

		for k, v := range addr_map {
			if k == self_ip || v.blocklisted {
				continue
			} else {
				send_request(k)
			}
		}

		mutex.Unlock()
	}
}

func server() {
	l, err := net.Listen("tcp", ":"+server_port)
	if err != nil {
		panic(err)
	}

	defer l.Close()

	for {
		conn, err := l.Accept()

		if err != nil {
			continue
		}

		if adversarial {
			go handle_request_adversarial(conn)
		} else {
			go handle_request(conn)
		}
	}
}

func handle_request(conn net.Conn) {
	defer conn.Close()

	for k, v := range addr_map {
		conn.Write([]byte(fmt.Sprintln(k + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(v.num))))
	}
}

func handle_request_adversarial(conn net.Conn) {
	defer conn.Close()

	tcp := conn.(*net.TCPConn)
	tcp.SetWriteBuffer(64)

	for k, v := range addr_map {
		tcp.Write([]byte(fmt.Sprintln(k + "," + fmt.Sprint(time.Now().Unix()+1) + "," + fmt.Sprint(v.num))))
		tcp.Write([]byte(fmt.Sprintln("127.0.0.1:" + strings.Split(k, ":")[1] + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(v.num))))
		tcp.Write([]byte(fmt.Sprintln("0.0.0.0:" + strings.Split(k, ":")[1] + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(v.num))))
		tcp.Write([]byte(fmt.Sprintln("8.8.8.8:80" + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(v.num))))
	}
}

package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
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
		conn, err := net.Dial("udp", "9.9.9.9:0")
		if err != nil {
			panic(err)
		}

		self_ip = conn.LocalAddr().(*net.UDPAddr).IP.String()

		conn.Close()
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

func send_request(addr string) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)

	mutex.Lock()
	defer mutex.Unlock()

	if err != nil {
		fmt.Println(err.Error())
		if entry, ok := addr_map[addr]; ok {
			(*entry).blocklisted = true
		}
	} else {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		update_addr_map(conn)
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
		mutex.RLock()

		for k, v := range addr_map {
			if k == self_ip || v.blocklisted {
				continue
			} else {
				go send_request(k)
				break
			}
		}

		mutex.RUnlock()
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

	mutex.RLock()
	defer mutex.RUnlock()

	for k, v := range addr_map {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		conn.Write([]byte(fmt.Sprintln(k + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(v.num))))
	}
}

func handle_request_adversarial(conn net.Conn) {
	defer conn.Close()

	tcp := conn.(*net.TCPConn)
	tcp.SetWriteBuffer(3)

	mutex.RLock()
	defer mutex.RUnlock()

	for k, v := range addr_map {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		tcp.Write([]byte(fmt.Sprintln(k + "," + fmt.Sprint(time.Now().Unix()+1) + "," + fmt.Sprint(v.num))))
		tcp.Write([]byte(fmt.Sprintln("127.0.0.1:" + strings.Split(k, ":")[1] + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(1))))
		tcp.Write([]byte(fmt.Sprintln("0.0.0.0:" + strings.Split(k, ":")[1] + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(0))))
		tcp.Write([]byte(fmt.Sprintln("8.8.8.8:80" + "," + fmt.Sprint(v.timestamp) + "," + fmt.Sprint(8))))
	}
}

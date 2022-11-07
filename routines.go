package main

import (
	"fmt"
	"net"
	"strings"
	"time"
)

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

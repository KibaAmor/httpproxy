package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	IP      string        `json:"ip"`
	Port    int           `json:"port"`
	Timeout time.Duration `json:"timeout"`
}

func (config *Config) GetAddr() string {
	return config.IP + ":" + strconv.Itoa(config.Port)
}

func (config *Config) GetTimeout() time.Duration {
	return config.Timeout * time.Second
}

func ReadConfig(path string) (config *Config, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return
	}
	config = &Config{}
	if err = json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	return
}

func main() {
	var config *Config

	if len(os.Args) == 1 {
		config = &Config{
			IP:      "0.0.0.0",
			Port:    8080,
			Timeout: 8,
		}
	} else {
		var err error
		config, err = ReadConfig(os.Args[1])
		if err != nil {
			log.Fatalf("read config [%s] failed: %v\n", os.Args[1], err)
		}
	}

	listener, err := net.Listen("tcp", config.GetAddr())
	if err != nil {
		log.Fatalf("listen failed: %v\n", err)
	}
	log.Println("listen at: " + config.GetAddr())
	log.Printf("timeout: %s", config.GetTimeout())

	for {
		client, err := listener.Accept()
		if err != nil {
			log.Fatalf("accept failed: %v\n", err)
		}
		go handleConn(client, config.GetTimeout())
	}
}

func handleConn(client net.Conn, timeout time.Duration) {
	defer client.Close()

	var buf [1024]byte
	n, err := client.Read(buf[:])
	if err != nil && err != io.EOF {
		log.Printf("read error: %v\n", err)
		return
	}

	data := strings.SplitN(string(buf[:n]), " ", 3)
	if len(data) >= 3 {
		parse, err := url.Parse(data[1])
		if err != nil {
			log.Printf("parse data[%s] failed: %v\n", data[1], err)
			return
		}

		var addr string
		if parse.Opaque == "443" {
			addr = parse.Scheme + ":443"
		} else {
			if strings.Contains(parse.Host, ":") {
				addr = parse.Host
			} else {
				addr = parse.Host + ":80"
			}
		}

		server, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			log.Printf("dail to [%s] failed: %v\n", addr, err)
			return
		}
		defer server.Close()

		if data[0] == "CONNECT" {
			client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		} else {
			server.Write(buf[:n])
		}
		go io.Copy(client, server)
		io.Copy(server, client)
	}
}

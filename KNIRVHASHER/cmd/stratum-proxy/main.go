package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

var (
	resultsFile *os.File
)

func main() {
	var err error
	resultsFile, err = os.OpenFile("stratum_shares.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer resultsFile.Close()

	fmt.Println("KNIRVHASHER Stratum Proxy starting on :3333")

	ln, err := net.Listen("tcp", ":3333")
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func log(format string, args ...interface{}) {
	fmt.Fprintf(resultsFile, "[%s] "+format+"\n", append([]interface{}{time.Now().Format(time.RFC3339Nano)}, args...)...)
	fmt.Printf("[%s] "+format+"\n", append([]interface{}{time.Now().Format(time.RFC3339Nano)}, args...)...)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	addr := conn.RemoteAddr().String()
	log("CONNECT: %s", addr)

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)

	go notifyLoop(encoder, addr)

	for {
		var raw json.RawMessage
		err := decoder.Decode(&raw)
		if err == io.EOF {
			log("DISCONNECT: %s", addr)
			return
		}
		if err != nil {
			return
		}

		var req map[string]interface{}
		json.Unmarshal(raw, &req)

		if method, ok := req["method"].(string); ok {
			switch method {
			case "mining.subscribe":
				resp := map[string]interface{}{
					"id": req["id"],
					"result": []interface{}{
						[]interface{}{
							[]interface{}{"mining.set_difficulty", "bd175551"},
							[]interface{}{"mining.notify", "4795c2d2"},
						},
						"08000000",
						2,
					},
					"error": nil,
				}
				encoder.Encode(resp)
				log("SUBSCRIBED: %s", addr)

			case "mining.authorize":
				resp := map[string]interface{}{
					"id":     req["id"],
					"result": true,
					"error":  nil,
				}
				encoder.Encode(resp)
				log("AUTHORIZED: %s", addr)

			case "mining.submit":
				if params, ok := req["params"].([]interface{}); ok {
					p, _ := json.Marshal(params)
					log("SHARE: %s", string(p))
				}
				resp := map[string]interface{}{
					"id":     req["id"],
					"result": true,
					"error":  nil,
				}
				encoder.Encode(resp)
			}
		}
	}
}

func notifyLoop(encoder *json.Encoder, addr string) {
	time.Sleep(100 * time.Millisecond)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		notify := map[string]interface{}{
			"method": "mining.notify",
			"params": []interface{}{
				fmt.Sprintf("%08x", time.Now().UnixNano() & 0xffffffff),
				"0000000000000000000000000000000000000000000000000000000000000000",
				"0100000001a4c7f3c5b0c8e0d2a9f7e4c5b0a3d2e1f7c5b0a3d2e1f7c5b0a3d2e1f00000000",
				"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
				"3ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a",
				"00000001",
				"170c8c6f",
				fmt.Sprintf("%08x", uint32(time.Now().Unix())),
				true,
			},
		}
		encoder.Encode(notify)
	}
}

package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type ConnectionPool struct {

	bufferChannelMap map[chan []byte]struct{}
	mu sync.Mutex

}

func(cp *ConnectionPool) AddConnection(bufferChannel chan []byte) {

	defer cp.mu.Unlock()
	cp.mu.Lock()

	cp.bufferChannelMap[bufferChannel] = struct{}{}

}

func(cp *ConnectionPool) DeleteConnection(bufferChannel chan []byte) {

	defer cp.mu.Unlock()
	cp.mu.Lock()

	delete(cp.bufferChannelMap, bufferChannel)

}

func(cp *ConnectionPool) Broadcast(buffer []byte) {

	defer cp.mu.Unlock()
	cp.mu.Lock()
	
	for bufferChannel, _ := range cp.bufferChannelMap {

		select {
			
		case bufferChannel <- buffer:

		default:


		}

	}		

}

func NewConnectionPool() *ConnectionPool {

	bufferChannelMap := make(map[chan []byte]struct{})
	return &ConnectionPool{bufferChannelMap: bufferChannelMap}

}

func main() {

	file, _ := os.Open(os.Args[1])
	ctn, err := io.ReadAll(file)
	if err != nil {
		
		log.Fatal(err)

	}
	
	connPool := NewConnectionPool()


	go func(connectionPool *ConnectionPool, content []byte) {
		
		buffer := make([]byte, 4096)

		for {
			
			// clear() is a new builtin function introduced in go 1.21
			clear(buffer)
			tempfile := bytes.NewReader(content)
			ticker := time.NewTicker(time.Millisecond * 250)

			for range ticker.C {
				
				_, err := tempfile.Read(buffer)

				if err == io.EOF {
					
					ticker.Stop()	
					break

				}
				
				connectionPool.Broadcast(buffer)
				
			}

		}

	}(connPool, ctn)


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		
		w.Header().Add("Content-Type","audio/aac")
		w.Header().Add("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			
			log.Println("Could not create flusher")

		}
			
		bufferChannel := make(chan []byte)
		connPool.AddConnection(bufferChannel)
		log.Printf("%s has connected\n", r.Host)

		for {
			

			buf := <-bufferChannel
			if _, err := w.Write(buf); err != nil {

				connPool.DeleteConnection(bufferChannel)
				log.Printf("%s's connection has been closed\n", r.Host)
				return

			}
			flusher.Flush()
			
		}


	})
	
	log.Println("Listening on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))

}



package main

import "log"

func main() {
	host := ":8080"
	s := NewTCPServer()
	if err := s.ListenAt(host); err != nil {
		log.Fatal(err)
	}

	log.Println("Listening at: ", host)
	s.Start()
}

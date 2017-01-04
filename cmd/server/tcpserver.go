package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/ihcsim/indexer"
)

// TCPServer can handle requests over TCP network.
type TCPServer struct {
	ln   net.Listener
	err  chan error
	quit chan os.Signal
	log  *log.Logger
	i    indexer.Indexer
}

// NewTCPServer returns an instance of TCPServer.
func NewTCPServer() *TCPServer {
	s := &TCPServer{
		err:  make(chan error),
		quit: make(chan os.Signal, 1),
		log:  log.New(os.Stdout, "", log.LstdFlags),
		i:    indexer.NewInMemoryIndexer(),
	}
	signal.Notify(s.quit, os.Interrupt)
	return s
}

// ListenAt sets up s to listen at host over TCP network.
func (s *TCPServer) ListenAt(host string) error {
	var err error
	s.ln, err = net.Listen("tcp", host)
	if err != nil {
		return err
	}

	return nil
}

// Start prepares s to handle incoming requests.
func (s *TCPServer) Start() {
	go s.catchSignals()
	s.acceptConn()
}

// Close issues a close message to the TCP listener of s.
func (s *TCPServer) Close() error {
	if s.ln != nil {
		return s.ln.Close()
	}

	close(s.err)
	close(s.quit)
	return nil
}

func (s *TCPServer) catchSignals() {
LOOP:
	for {
		select {
		case <-s.quit:
			s.log.Println("Shutting down server")
			if err := s.Close(); err != nil {
				s.err <- err
			}
			break LOOP
		case e := <-s.err:
			s.log.Println("Error in handleConn():", e)
		}
	}
}

func (s *TCPServer) acceptConn() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return err
		}

		go s.handleConn(conn)
	}
	s.Close()
	return nil
}

func (s *TCPServer) handleConn(conn net.Conn) {
	defer conn.Close()

	for {
		line, err := s.read(conn)
		if err != nil {
			if err == io.EOF {
				break
			}
			s.err <- fmt.Errorf("Reader Error: %s", err)
			continue
		}
		s.log.Printf("[RECV] %s (%d bytes): %s", conn.RemoteAddr().String(), len(line), line)

		res := s.process(line)

		if err := s.write(conn, res); err != nil {
			if err == io.EOF {
				break
			}
			s.err <- fmt.Errorf("Writer Error: %s", err)
			continue
		}
		s.log.Printf("[SEND] %s (%d bytes): %s", conn.RemoteAddr().String(), len(res), res)
	}
}

func (s *TCPServer) read(conn net.Conn) (string, error) {
	r := bufio.NewReader(conn)
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return line, nil
}

func (s *TCPServer) process(line string) string {
	if pkg, cmd, err := indexer.ParseMsg(line); err != nil {
		s.err <- err
		return indexer.Error
	} else {
		switch cmd {
		case "INDEX":
			return s.i.Index(pkg)
		case "REMOVE":
			return s.i.Remove(pkg.Name)
		case "QUERY":
			return s.i.Query(pkg.Name)
		default:
			return indexer.Error
		}
	}
}

func (s *TCPServer) write(conn net.Conn, res string) error {
	w := bufio.NewWriter(conn)
	if _, err := w.WriteString(res); err != nil {
		return err
	}
	w.Flush()
	return nil
}

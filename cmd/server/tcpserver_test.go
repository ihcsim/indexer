package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ihcsim/indexer"
)

func TestListenAt(t *testing.T) {
	s := NewTCPServer()
	defer s.Close()

	host := randomLocalPort()
	if err := s.ListenAt(host); err != nil {
		t.Error("Expected server to listen successfully at", host)
	}

	if err := s.Close(); err != nil {
		t.Error("Unexpected error while closing server:", err)
	}
}

func TestCatchSignals(t *testing.T) {
	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	s := NewTCPServer()
	s.log.SetOutput(w)
	go s.catchSignals()

	// expect error message to be captured
	errMsg := "test error message"
	s.err <- fmt.Errorf(errMsg)
	log := make([]byte, 256)
	if _, err := r.Read(log); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), errMsg) {
		t.Errorf("Expected log to capture error %q, but got %q", errMsg, string(log))
	}

	// expected shutdown message to be captured
	s.quit <- os.Interrupt
	quitMsg := "Shutting down server"
	log = make([]byte, 256)
	if _, err := r.Read(log); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), quitMsg) {
		t.Error("Expected shutdown log to be captured, but got", string(log))
	}
}

func TestRead(t *testing.T) {
	s := NewTCPServer()
	defer s.Close()

	if err := s.ListenAt(randomLocalPort()); err != nil {
		t.Fatal(err)
	}

	// perform read on a new connection
	expected := "This is a test message\n"
	go func() {
		conn, err := s.ln.Accept()
		if err != nil {
			t.Error(err)
		}

		actual, err := s.read(conn)
		if err != nil {
			t.Error(err)
		}

		if expected != actual {
			t.Errorf("Expected message read to be %q, but got %q", expected, actual)
		}
	}()

	client, err := tcpClient(s.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	client.Write([]byte(expected))
}

func TestWrite(t *testing.T) {
	s := NewTCPServer()
	defer s.Close()

	if err := s.ListenAt(randomLocalPort()); err != nil {
		t.Fatal(err)
	}

	// perform write on a new connection
	expected := "This is a test message\n"
	go func() {
		conn, err := s.ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		if err := s.write(conn, expected); err != nil {
			t.Error("Unexpected error duing write:", err)
		}
	}()

	client, err := tcpClient(s.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	b := make([]byte, len(expected))
	if _, err := client.Read(b); err != nil {
		t.Error(err)
	}
	if expected != string(b) {
		t.Errorf("Expected message received to be %q, but got %q", expected, string(b))
	}
}

func TestProcess(t *testing.T) {
	s := NewTCPServer()
	defer s.Close()

	s.i = &MockIndexer{}
	if err := s.ListenAt(randomLocalPort()); err != nil {
		t.Error(err)
	}

	var tests = []struct {
		msg      string
		expected string
	}{
		{msg: "INDEX|ccng|libcurl\n", expected: indexer.OK},
		{msg: "REMOVE|ccng|libcurl\n", expected: indexer.OK},
		{msg: "QUERY|ccng|libcurl\n", expected: indexer.OK},
		{msg: "UNKNOWN|ccng|libcurl\n", expected: indexer.Error},
	}

	for _, test := range tests {
		actual := s.process(test.msg)
		if actual != test.expected {
			t.Errorf("Expected response for msg %q to be %q, but got %q", test.msg, test.expected, actual)
		}
	}
}

func TestProcess_Error(t *testing.T) {
	s := NewTCPServer()
	defer s.Close()

	var tests = []struct {
		msg      string
		expected string
	}{
		{msg: "", expected: indexer.ErrMalformedMsg},
		{msg: "|ccng|libcurl\n", expected: indexer.ErrMissingCmd},
		{msg: "INDEX||libcurl\n", expected: indexer.ErrMissingName},
	}

	// capture error from server
	ready := make(chan struct{})
	var err error
	go func() {
		for {
			err = <-s.err
			ready <- struct{}{}
		}
	}()

	// call the process method for each test to trigger an error
	for _, test := range tests {
		res := s.process(test.msg)

		<-ready
		if res != indexer.Error {
			t.Errorf("Expected response to be %q, but got %q", indexer.Error, res)
		}

		if fmt.Sprintf("%s", err) != test.expected {
			t.Errorf("Expected error message to be %q, but got %q", test.expected, err)
		}
	}
}

func TestHandleConn(t *testing.T) {
	s := NewTCPServer()
	defer s.Close()
	s.log.SetOutput(ioutil.Discard)

	if err := s.ListenAt(randomLocalPort()); err != nil {
		t.Fatal(err)
	}

	// prepare to accept client connection
	go func() {
		if err := s.acceptConn(); err != nil {
			t.Error(err)
		}
	}()

	// ensure server is fully started
	time.Sleep(time.Millisecond * 10)

	client, err := tcpClient(s.ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	// send messages to server
	msgs := []string{"REMOVE|ccng|\n", "INDEX|ccng|\n", "QUERY|ccng|\n"}
	for _, msg := range msgs {
		if _, err := client.Write([]byte(msg)); err != nil {
			t.Fatal(err)
		}

		// wait for server to respond
		time.Sleep(time.Millisecond * 200)

		res := make([]byte, len(indexer.OK))
		if _, err := client.Read(res); err != nil {
			t.Fatal(err)
		}
		if string(res) != indexer.OK {
			t.Errorf("Expect response to be OK, but got %q", string(res))
		}
	}
	client.Close()
}

func tcpClient(host string) (net.Conn, error) {
	return net.DialTimeout("tcp", host, time.Millisecond*10)
}

func randomLocalPort() string {
	port := rand.Intn(49152) + 16383
	return ":" + strconv.Itoa(port)
}

// MockIndexer mocks out the Indexer interfaces for testing purposes.
type MockIndexer struct{}

func (m *MockIndexer) Index(p *indexer.Pkg) string {
	return indexer.OK
}

func (m *MockIndexer) Remove(name string) string {
	return indexer.OK
}

func (m *MockIndexer) Query(name string) string {
	return indexer.OK
}

package httpapi

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type nilService struct{ Service }

func TestJSONReadDeadline(t *testing.T) {
	old := jsonReadTimeout
	jsonReadTimeout = 100 * time.Millisecond
	t.Cleanup(func() { jsonReadTimeout = old })

	srv := httptest.NewServer(NewHandler(nilService{}, Options{}))
	t.Cleanup(srv.Close)

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	// Promise 100 bytes, send one, then stall.
	_, err = conn.Write([]byte("POST /api/setup HTTP/1.1\r\nHost: x\r\nContent-Type: application/json\r\n" +
		"Content-Length: 100\r\n\r\n{"))
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	start := time.Now()
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestTimeout {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Fatalf("took %v", d)
	}
}

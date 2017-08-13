package tunnel

import (
	"bytes"
	"fmt"
	stdlog "log"
	"net"
	"os"
	"sync"
	"testing"

	logging "github.com/op/go-logging"
	"github.com/shell909090/goproxy/sutils"
)

const (
	PAYLOAD = "foobar"
)

func echo_client(t *testing.T, conn net.Conn, wg *sync.WaitGroup) {
	defer conn.Close()
	defer wg.Done()

	for i := 0; i < 100; i++ {
		var buf bytes.Buffer

		_, err := buf.WriteString(PAYLOAD)
		if err != nil {
			t.Error(err)
			return
		}

		_, err = fmt.Fprintf(&buf, "%d", i)
		if err != nil {
			t.Error(err)
			return
		}

		b := buf.Bytes()

		n, err := conn.Write(b)
		if err != nil {
			return
		}
		if n < len(b) {
			t.Error("short write")
			return
		}

		var readbuf [100]byte
		n, err = conn.Read(readbuf[:])
		if err != nil {
			return
		}
		if bytes.Compare(b, readbuf[:n]) != 0 {
			t.Error("data not match")
			return
		}
	}
}

func multi_client(t *testing.T, client *Client, wg *sync.WaitGroup) {
	for i := 0; i < 10; i++ {
		conn, err := client.Dial("tcp", "127.0.0.1:14756")
		if err != nil {
			t.Error(err)
			return
		}

		wg.Add(1)
		go echo_client(t, conn, wg)
	}
}

func SetLogging() {
	logBackend := logging.NewLogBackend(os.Stderr, "",
		stdlog.Ltime|stdlog.Lmicroseconds|stdlog.Lshortfile)
	logging.SetBackend(logBackend)
	logging.SetFormatter(
		logging.MustStringFormatter("%{module}[%{level}]: %{message}"))
	lv, _ := logging.LogLevel("INFO")
	logging.SetLevel(lv, "")
	return
}

func TestTunnel(t *testing.T) {
	var wg sync.WaitGroup
	SetLogging()

	wg.Add(2)
	go sutils.EchoServer(&wg)
	go func() {
		err := RunMockServer(&wg)
		if err != nil {
			t.Error(err)
		}
		return
	}()
	wg.Wait()

	dc := NewDialerCreator(sutils.DefaultTcpDialer, "tcp4", "127.0.0.1:14755", "", "")

	client, err := dc.Create()
	if err != nil {
		t.Error(err)
		return
	}
	go func() {
		client.Loop()
		logger.Warning("client loop quit")
	}()

	multi_client(t, client, &wg)
	wg.Wait()

	multi_client(t, client, &wg)
	client.Close()
	wg.Wait()
}
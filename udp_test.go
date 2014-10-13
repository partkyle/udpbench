package udp

import (
	"bufio"
	"fmt"
	"net"
	"syscall"
	"testing"
)

var testUDPMessage = "testing writing over udp"

type Fatality interface {
	Fatal(...interface{})
}

func listenUdp(fatal Fatality) *net.UDPConn {
	addr := net.UDPAddr{Port: 0, IP: net.ParseIP("127.0.0.1")}
	conn, err := net.ListenUDP("udp", &addr)

	if err != nil {
		fatal.Fatal(err)
	}
	return conn
}

func BenchmarkDialEachTime(t *testing.B) {
	server := listenUdp(t)

	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		conn, err := net.Dial("udp", server.LocalAddr().String())
		if err != nil {
			t.Fatal(err)
		}

		_, err = fmt.Fprintln(conn, testUDPMessage)
		if err != nil {
			t.Fatal(err)
		}

		conn.Close()
	}

	t.StopTimer()

	server.Close()
}

func BenchmarkDialOnce(b *testing.B) {
	server := listenUdp(b)

	conn, err := net.Dial("udp", server.LocalAddr().String())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fmt.Fprintln(conn, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}
	}

	conn.Close()

	b.StopTimer()

	server.Close()
}

func BenchmarkBuffered(b *testing.B) {
	server := listenUdp(b)

	conn, err := net.Dial("udp", server.LocalAddr().String())
	if err != nil {
		b.Fatal(err)
	}

	buf := bufio.NewWriterSize(conn, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fmt.Fprintln(buf, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}
	}

	buf.Flush()
	conn.Close()

	b.StopTimer()

	server.Close()
}

func BenchmarkSyscall(b *testing.B) {
	server := listenUdp(b)

	b.ResetTimer()

	raddr := syscall.SockaddrInet4{Port: 8125, Addr: [4]byte{0, 0, 0, 0}}

	for i := 0; i < b.N; i++ {
		fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.PROT_NONE)
		if err != nil {
			b.Fatal(err)
		}

		err = syscall.Sendmsg(fd, []byte(testUDPMessage), nil, &raddr, syscall.MSG_DONTWAIT)
		if err != nil {
			b.Fatal(err)
		}

		syscall.Close(fd)
	}

	server.Close()
}

func BenchmarkPersistantSyscall(b *testing.B) {
	server := listenUdp(b)

	b.ResetTimer()

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.PROT_NONE)

	if err != nil {
		b.Fatal(err)
	}

	raddr := syscall.SockaddrInet4{Port: 8125, Addr: [4]byte{0, 0, 0, 0}}

	for i := 0; i < b.N; i++ {
		err = syscall.Sendmsg(fd, []byte(testUDPMessage), []byte{}, &raddr, syscall.MSG_DONTWAIT)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.StopTimer()

	server.Close()
}

type syscallWrapper struct {
	fd   int
	addr syscall.Sockaddr
}

func (s *syscallWrapper) Write(b []byte) (int, error) {
	return len(b), syscall.Sendmsg(s.fd, b, nil, s.addr, syscall.MSG_DONTWAIT)
}

func (s *syscallWrapper) Close() error {
	return syscall.Close(s.fd)
}

func BenchmarkSyscallConnWrapper(b *testing.B) {
	server := listenUdp(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.PROT_NONE)
		if err != nil {
			b.Fatal(err)
		}

		raddr := syscall.SockaddrInet4{Port: 8125, Addr: [4]byte{0, 0, 0, 0}}

		conn := &syscallWrapper{fd: fd, addr: &raddr}

		_, err = fmt.Fprintln(conn, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}

		conn.Close()
	}

	b.StopTimer()

	server.Close()
}

func BenchmarkSyscallConnWrapperPersistant(b *testing.B) {
	server := listenUdp(b)

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.PROT_NONE)

	if err != nil {
		b.Fatal(err)
	}

	raddr := &syscall.SockaddrInet4{Port: 8125, Addr: [4]byte{0, 0, 0, 0}}

	conn := &syscallWrapper{fd: fd, addr: raddr}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fmt.Fprintln(conn, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}
	}

	conn.Close()

	b.StopTimer()

	server.Close()
}

func BenchmarkSyscallConnWrapperBuffered(b *testing.B) {
	server := listenUdp(b)

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.PROT_NONE)

	if err != nil {
		b.Fatal(err)
	}

	raddr := &syscall.SockaddrInet4{Port: 8125, Addr: [4]byte{0, 0, 0, 0}}

	conn := &syscallWrapper{fd: fd, addr: raddr}
	buf := bufio.NewWriterSize(conn, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := fmt.Fprintln(buf, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}
	}

	buf.Flush()
	conn.Close()

	b.StopTimer()

	server.Close()
}

type writeToConn struct {
	packetListener net.PacketConn
	remoteAddr     *net.UDPAddr
}

func newWriteToConn(raddr string) (*writeToConn, error) {
	// need to create the local socket for sending messages from
	packetListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	// resolve the udp address provided
	remoteAddr, err := net.ResolveUDPAddr("udp", raddr)
	if err != nil {
		return nil, err
	}

	return &writeToConn{packetListener: packetListener, remoteAddr: remoteAddr}, nil
}

func (w *writeToConn) Write(p []byte) (int, error) {
	return w.packetListener.WriteTo(p, w.remoteAddr)
}

func (w *writeToConn) Close() error {
	return w.packetListener.Close()
}

func BenchmarkWriteTo(b *testing.B) {
	server := listenUdp(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn, err := newWriteToConn(server.LocalAddr().String())
		if err != nil {
			b.Fatal(err)
		}

		_, err = fmt.Fprintln(conn, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()
	}

	b.StopTimer()

	server.Close()
}

func BenchmarkWriteToPersistant(b *testing.B) {
	server := listenUdp(b)

	conn, err := newWriteToConn(server.LocalAddr().String())

	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = fmt.Fprintln(conn, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}
	}

	conn.Close()

	b.StopTimer()

	server.Close()
}

func BenchmarkWriteToPersistantBuffered(b *testing.B) {
	server := listenUdp(b)

	conn, err := newWriteToConn(server.LocalAddr().String())
	if err != nil {
		b.Fatal(err)
	}

	buf := bufio.NewWriterSize(conn, 512)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = fmt.Fprintln(buf, testUDPMessage)
		if err != nil {
			b.Fatal(err)
		}
	}

	buf.Flush()
	conn.Close()

	b.StopTimer()

	server.Close()
}

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

var (
	listenPort  int
	targetAddrS string
	targetAddr  *net.TCPAddr

	id      = 0
	logPath = "./" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "/"
)

func init() {
	flag.IntVar(&listenPort, "p", 0, "listen port")
	flag.StringVar(&targetAddrS, "t", "", "target addr")
	flag.Parse()
}

func main() {
	if listenPort == 0 {
		exitError(fmt.Errorf("place input listen port: -p"))
	}
	if targetAddrS == "" {
		exitError(fmt.Errorf("place input target addr: -t"))
	}

	listenAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		exitError(err)
	}

	targetAddr, err = net.ResolveTCPAddr("tcp", targetAddrS)
	if err != nil {
		exitError(err)
	}

	listen, err := net.ListenTCP("tcp", listenAddr)
	if err != nil {
		exitError(err)
	}

	fmt.Println("logPath: ", logPath)
	err = os.MkdirAll(logPath, 0700)
	if err != nil {
		exitError(err)
	}
	for {
		clientConn, err := listen.AcceptTCP()
		if err != nil {
			exitError(err)
		}
		id++
		go dump(clientConn, id)
	}
}

func dump(clientConn *net.TCPConn, id int) {
	defer func() {
		_ = clientConn.Close()
	}()

	serverConn, err := net.DialTCP("tcp", nil, targetAddr)
	if err != nil {
		println(id, "dial target error", err.Error())
		return
	}
	defer func() {
		_ = serverConn.Close()
	}()

	c2sFile, err := os.Create(fmt.Sprintf("%s%d.c2s.dump", logPath, id))
	if err != nil {
		println("open file error", err.Error())
		return
	}
	defer func() {
		_ = c2sFile.Close()
	}()
	s2cFile, err := os.Create(fmt.Sprintf("%s%d.s2c.dump", logPath, id))
	if err != nil {
		println("open file error", err.Error())
		return
	}
	defer func() {
		_ = s2cFile.Close()
	}()

	wait := make(chan int, 2)
	go doDumpC2S(clientConn, serverConn, c2sFile, wait, id)
	go doDumpS2C(serverConn, clientConn, s2cFile, wait, id)

	_ = <-wait
	_ = <-wait
}

func doDumpC2S(c1 *net.TCPConn, c2 *net.TCPConn, file *os.File, wait chan int, id int) {
	defer func() {
		wait <- 1
	}()
	buf := make([]byte, 64*1024)
	for {
		readLen, err := c1.Read(buf)
		if err != nil {
			break
		}
		_, _ = file.Write([]byte(fmt.Sprintf("time:\n%s\ndata:\n", time.Now().String())))
		_, _ = file.Write(buf[:readLen])
		_, _ = file.Write([]byte("\n"))
		_ = file.Sync()
		if _, err = c2.Write(buf[:readLen]); err != nil {
			break
		}
	}
}

var first = true

func doDumpS2C(c1 *net.TCPConn, c2 *net.TCPConn, file *os.File, wait chan int, id int) {
	defer func() {
		wait <- 1
	}()
	buf := make([]byte, 64*1024)
	for i := 1; ; i++ {
		readLen, err := c1.Read(buf)
		if err != nil {
			break
		}
		if id == 2 && i == 2 {
			continue
		}
		if id == 3 && i == 3 {
			continue
		}
		_, _ = file.Write([]byte(fmt.Sprintf("time:\n%s\ndata:\n", time.Now().String())))
		_, _ = file.Write(buf[:readLen])
		_, _ = file.Write([]byte("\n"))
		_ = file.Sync()
		if _, err = c2.Write(buf[:readLen]); err != nil {
			break
		}
	}
}

func exitError(err error) {
	println("error: ", err.Error())
	println("place exit")
	buf := make([]byte, 1)
	_, _ = os.Stdin.Read(buf)
	os.Exit(0)
}

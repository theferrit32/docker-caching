package main

import (
	"fmt"
	"io"
	"net"
	"strings"
)

func handleConnection(conn net.Conn, c chan string, destSockPath string) {
	client, err := net.Dial("unix", destSockPath)
	if err != nil {
		logger.Errorf("failed to connect to dest socket")
		return
	}
	var str strings.Builder
	for {
		buf := make([]byte, 512)
		nr, err := conn.Read(buf)
		if err != nil {
			logger.Infof("could not read bytes")
			break
		}
		if nr == 0 {
			break
		}
		str.WriteString(string(buf[0:nr]))
		http_end := []byte{'\r', '\n', '\r', '\n'}
		s := str.String()
		if len(s) > 4 {
			last_4 := []byte{
				s[len(s)-4],
				s[len(s)-3],
				s[len(s)-2],
				s[len(s)-1]}
			fmt.Printf("last two %02x %02x\n", last_4[0], last_4[1])
			if string(last_4) == string(http_end) {
				logger.Infof("end of http request")
				break
			}
		}
	}
	fmt.Printf("str: %s", str.String())
	logger.Infof("finished reading\n")
	client.Write([]byte(str.String()))
	logger.Infof("write to dest")
	var response strings.Builder
	respBuf := make([]byte, 512)
	for {
		rc, err := client.Read(respBuf)
		if err == io.EOF {
			break
		} else {
			logger.Errorf("error reading from destination")
		}
		response.Write(respBuf[0:rc])
	}
	c <- response.String()
}

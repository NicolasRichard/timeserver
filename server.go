package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {

	app := &cli.App{
		Name: "Time Server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "port",
				Value: "8000",
				Usage: "port on which to serve requests",
			},
		},
		Action: func(ctx *cli.Context) error {
			return start(":" + ctx.String("port"))
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func start(port string) error {
	var wg sync.WaitGroup
	wg.Add(2)

	localUDPAddr, error := net.ResolveUDPAddr("udp", port)
	if error != nil {
		return error
	}

	udpConnection, error := net.ListenUDP("udp", localUDPAddr)
	if error != nil {
		return error
	}

	defer udpConnection.Close()
	go startUDP(udpConnection, localUDPAddr, &wg)

	localTCPAddr, error := net.ResolveTCPAddr("tcp", port)
	if error != nil {
		return error
	}

	tcpListener, error := net.ListenTCP("tcp", localTCPAddr)
	if error != nil {
		return error
	}

	defer tcpListener.Close()
	go startTCP(tcpListener, localTCPAddr, &wg)

	wg.Wait()
	return nil
}

func startUDP(connection *net.UDPConn, localAddr *net.UDPAddr, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		buffer := make([]byte, 1024)
		packetSize, resquestAddr, error := connection.ReadFromUDP(buffer)
		if error != nil {
			fmt.Println("Unable to read packet.")
			continue
		}

		go receiveUDPMessage(buffer[0:packetSize], connection, resquestAddr)
	}
}

func startTCP(listener *net.TCPListener, localAddr *net.TCPAddr, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		connection, error := listener.Accept()
		if error != nil {
			fmt.Println("Error while waiting for a TCP connection.")
		}

		go receiveTCPMessage(connection)
	}
}

func receiveUDPMessage(packet []byte, connection *net.UDPConn, addr *net.UDPAddr) {
	currentTime := process(string(packet))
	connection.WriteToUDP([]byte(currentTime), addr)
}

func receiveTCPMessage(connection net.Conn) {
	defer connection.Close()

	reader := bufio.NewReader(connection)
	for {
		connection.SetDeadline(time.Now().Add(time.Second * 5))
		command, error := reader.ReadString('\n')
		if neterror, ok := error.(net.Error); ok && neterror.Timeout() {
			fmt.Println("TCP timeout")
			return
		} else if error == io.EOF {
			return
		} else if error != nil {
			fmt.Println("Unknown error")
			return
		}

		currentTime := process(strings.TrimSuffix(command, "\n"))
		connection.Write([]byte(currentTime))
	}
}

func process(command string) string {
	var result string

	switch command {
	case "date":
		result = time.Now().Format("2006-01-02")
	case "time":
		result = time.Now().Format("15:04:05Z07:00")
	case "datetime":
		result = time.Now().Format(time.RFC3339)
	default:
		result = "Error: unknown command"
	}

	return result + "\n"
}

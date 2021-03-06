package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/peer-calls/peer-calls/server"
	"github.com/peer-calls/peer-calls/server/logger"
)

var gitDescribe string = "v0.0.0"

func fileWalk() {
	fmt.Println("FILEEEEE")
	files, err := ioutil.ReadDir("../")
	if err != nil {
		fmt.Println("ERROR 1")

		log.Fatal(err)
	}

	for _, f := range files {
		fmt.Println("../" + f.Name())
		if f.IsDir() {
			filesI, errI := ioutil.ReadDir("../" + f.Name())
			if errI != nil {
				fmt.Println("ERROR INTERALL")
				fmt.Print(err)
				log.Fatal(err)
			}
			for _, fI := range filesI {
				fmt.Println(f.Name() + "/" + fI.Name())
				if f.IsDir() {
					fmt.Println("IS A DIRECTORY")

				} else {
					fmt.Println("IS A FILE")

				}

			}
		} else {
			fmt.Println("ITS A FILEEE")
		}
	}

	fmt.Println("FILEEEEEES #2")

}
func configure(loggerFactory *logger.Factory, args []string) (net.Listener, *server.StartStopper, error) {
	log.Println("Starting CONFIG V2222")
	//fileWalk()

	log := loggerFactory.GetLogger("main")

	flags := flag.NewFlagSet("peer-calls", flag.ExitOnError)
	var configFilename string
	flags.StringVar(&configFilename, "c", "", "Config file to use")
	flags.Parse(args)

	configFiles := []string{}
	if configFilename != "" {
		configFiles = append(configFiles, configFilename)
	}
	c, err := server.ReadConfig(configFiles)
	if err != nil {
		return nil, nil, fmt.Errorf("Error reading config: %w", err)
	}

	log.Printf("Using config: %+v", c)
	newAdapter := server.NewAdapterFactory(loggerFactory, c.Store)
	rooms := server.NewAdapterRoomManager(newAdapter.NewAdapter)
	tracks := server.NewMemoryTracksManager(loggerFactory, c.Network.SFU.JitterBuffer)
	mux := server.NewMux(loggerFactory, c.BaseURL, gitDescribe, c.Network, c.ICEServers, rooms, tracks, c.Prometheus)
	l, err := net.Listen("tcp", net.JoinHostPort(c.BindHost, strconv.Itoa(c.BindPort)))
	if err != nil {
		return nil, nil, fmt.Errorf("Error starting server listener: %w", err)
	}
	startStopper := server.NewStartStopper(server.ServerParams{
		TLSCertFile: c.TLS.Cert,
		TLSKeyFile:  c.TLS.Key,
		// TLSCertFile: "./config/cert.pem",
		// TLSKeyFile:  "./config/key.pem",
	}, mux)
	log.Println("ENDING CONFIG")

	return l, startStopper, nil
}

func start(args []string) (addr *net.TCPAddr, stop func() error, errChan <-chan error) {
	loggerFactory := logger.NewFactoryFromEnv("PEERCALLS_", os.Stderr)
	loggerFactory.SetDefaultEnabled([]string{
		"-sdp",
		"-ws",
		"-nack",
		"-rtp",
		"-rtcp",
		"-pion:*:trace",
		"-pion:*:debug",
		"-pion:*:info",
		"*",
	})
	log := loggerFactory.GetLogger("main")

	ch := make(chan error, 1)
	log.Println("ABOUT TO CONFIGURE")
	l, startStopper, err := configure(loggerFactory, args)
	if err != nil {
		ch <- err
		close(ch)
		return nil, nil, ch
	}
	addr = l.Addr().(*net.TCPAddr)
	log.Printf("Listening on: %s", addr.String())
	go func() {
		err := startStopper.Start(l)
		if err != http.ErrServerClosed {
			ch <- fmt.Errorf("Error starting server: %w", err)
		} else {
			ch <- nil
		}
		close(ch)
	}()
	return addr, startStopper.Stop, ch
}

func main() {
	_, _, errChan := start(os.Args[1:])
	err := <-errChan
	if err != nil {
		fmt.Println("Error starting server: %w", err)
		os.Exit(1)
	}
}

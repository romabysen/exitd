package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

var wg sync.WaitGroup

type writer struct {
	io.Writer
	logname string
}

func (w writer) Write(b []byte) (n int, err error) {
	s := []string{time.Now().Format("2006/01/02 15:04:05 "), w.logname, ": "}
	return w.Writer.Write(append([]byte(strings.Join(s, "")), b...))
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 2 {
		log.Fatal("Need at least two programs to run.")
	}

	quitchan := make(chan string)
	sigchan := make(chan os.Signal, 1)
	procs := make([]*os.Process, 0)
	gotSignal := false

	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	for index, command := range flag.Args() {
		proc := startCommand(index, command, quitchan)
		procs = append(procs, proc)
		wg.Add(1)
	}

	select {
	case msg := <-quitchan:
		log.Printf("Command \"%s\" exited.", msg)
	case sig := <-sigchan:
		gotSignal = true
		log.Printf("Got signal: %s", sig)
	}

	for _, proc := range procs {
		proc.Signal(syscall.SIGTERM)
	}
	log.Println("Waiting for remaining processes to exit...")
	if waitTimeout(&wg, 5*time.Second) {
		log.Println("Some processes did not exit in time.")
	}

	if gotSignal {
		log.Println("Done")
	} else {
		log.Fatalln("Done. One or more processes exited prematurely.")
	}
}

func startCommand(index int, command string, quitchan chan<- string) *os.Process {
	s := strings.Split(command, "/")
	logname := s[len(s)-1]
	var logger *log.Logger

	noPrefix := os.Getenv("EXITD_NO_LOG_PREFIX")
	if noPrefix == "true" {
		logger = log.New(os.Stdout, "", 0)
	} else if noPrefix == "first" {
		if index == 0 {
			logger = log.New(os.Stdout, "", 0)
		} else {
			logger = log.New(&writer{os.Stdout, logname}, "", 0)
		}
	} else {
		logger = log.New(&writer{os.Stdout, logname}, "", 0)
	}

	cmd := exec.Command(command)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	combined := io.MultiReader(stdout, stderr)

	if err = cmd.Start(); err != nil {
		panic(err)
	}

	outbuf := bufio.NewScanner(combined)
	go func() {
		for outbuf.Scan() {
			logger.Println(outbuf.Text())
		}
		wg.Done()
		quitchan <- logname
	}()
	return cmd.Process
}

// waitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
// From https://gist.github.com/x32net/b060828f9e1be671b4c94036ea9ef553
func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
}

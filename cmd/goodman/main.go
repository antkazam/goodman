package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/snikch/goodman"
)

var c chan os.Signal
var cmds chan *exec.Cmd
var runners []goodman.Runner
var hookServerInitalPort = 61322
var hooksServerCount int

func main() {
	cmds = make(chan *exec.Cmd, 50)
	args := os.Args
	hookPaths := args[1:len(args)]
	c = make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		closeHooksServers()
		os.Exit(0)
	}()
	hooksServerCount = len(args) - 1
	if len(args) < 2 {
		runners = append(runners, &goodman.DummyRunner{})
	} else {
		for _, path := range hookPaths {
			cmd := exec.Command(path, fmt.Sprintf("-port=%d", hookServerInitalPort))
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Println("Sending to channel\n")
			cmds <- cmd
			fmt.Println("Completed")
			go func() {
				log.Printf("Starting hooks server in go routine")
				err := cmd.Run()
				if err != nil {
					fmt.Println("Hooks client failed with " + err.Error())
				}
			}()
			// Must sleep so go routine running hooks server has chance to startup
			time.Sleep(100 * time.Millisecond)
			runners = append(runners, goodman.NewRunner("Hooks", hookServerInitalPort))
			hookServerInitalPort++
		}
	}
	close(cmds)
	server := goodman.NewServer(runners)
	err := server.Run()
	if err != nil {
		log.Fatal(err.Error())
	}
	closeHooksServers()
}

func closeHooksServers() {
	log.Printf("Shutting down hooks servers\n")
	count := 0
	for cmd := range cmds {
		cmd.Process.Kill()
		count++
		if hooksServerCount == count {
			return
		}
	}
}

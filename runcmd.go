/*
	Tool runcmd is used for running concurrently any commands on remote servers.
	Ssh is used as a transport for remote communications, so you must have a key-based
	access from where you run "runcmd" to the remote servers.
	Usage:
		-c string
        		Command to execute in quotes if needed (default "uptime")
 		 -f string
        		File with servers list (one per line); comments using # are allowed
  		-r int
        		Number of concurrently running servers; must be used with -t parameter
  		-s string
       			List of servers in quotes
  		-t int
        		Time to sleep in seconds before to  concurently run a next group of servers; must be used with -r parameter
*/
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func getAnswer(server string, cmd string) ([]byte, error) {
	resp, err := exec.Command("ssh", server, cmd).Output()
	if err != nil {
		log.Fatal(err)
	}

	return resp, nil
}

func worker(srvCh chan string, cmdCh chan string, respCh chan string) {
	for {
		server := <-srvCh
		cmd := <-cmdCh
		resp, err := getAnswer(server, cmd)
		if err == nil {
			respCh <- fmt.Sprintf("%s result: %s\n", server, resp)
		} else {
			respCh <- fmt.Sprintf("Error processing %s: %s", server, err)
		}
	}
}

func generator(server, cmd string, srvCh chan string, cmdCh chan string) {
	srvCh <- server
	cmdCh <- cmd
}

func readHosts(f string) (servers []string) {
	dat, err := ioutil.ReadFile(f)
	if err != nil {
		log.Fatal(err)
	}
	s := string(dat)
	s = strings.TrimRight(s, "\n")
	servers = strings.Split(s, "\n")
	return
}

func main() {

	var servers []string

	respCh := make(chan string)
	srvCh := make(chan string)
	cmdCh := make(chan string)

	srvlist := flag.String("s", "", "List of servers in quotes")
	fpath := flag.String("f", "", "File with servers list (one per line); comments using # are allowed")
	srvrun := flag.Int("r", 0, "Number of concurrently running servers; must be used with -t parameter")
	sleeptime := flag.Int("t", 0, "Time to sleep in seconds before to  concurently run a next group of servers; must be used with -r parameter")
	cmd := flag.String("c", "uptime", "Command to execute in quotes if needed")

	flag.Parse()

	if len(os.Args[1:]) == 0 {
		flag.Usage()
		os.Exit(0)
	}
	if *srvlist != "" && *fpath != "" {
		fmt.Println("Please provide only one parameter at time: -s or -f.")
		os.Exit(0)
	}

	if (*srvrun == 0 && *sleeptime != 0) || (*srvrun != 0 && *sleeptime == 0) {
		fmt.Println("Please provide two parameters simultaneously: -r and -t.")
		os.Exit(0)
	}

	if *srvlist != "" {
		servers = strings.Split(*srvlist, " ")
	}

	if *fpath != "" {
		servers = readHosts(*fpath)
	}

	srvcount := len(servers)
	counter := 1

	for _, server := range servers {
		go generator(server, *cmd, srvCh, cmdCh)
	}

	for i := 0; i < srvcount; i++ {
		go worker(srvCh, cmdCh, respCh)
		if *srvrun != 0 && counter%*srvrun == 0 {
			time.Sleep(time.Duration(*sleeptime) * time.Second)
		}
		if *srvrun != 0 {
			fmt.Print(<-respCh)
		}
		counter++
	}
	if *srvrun == 0 {
		for i := 0; i < srvcount; i++ {
			fmt.Print(<-respCh)
		}
	}

}

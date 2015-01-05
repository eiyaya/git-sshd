package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/crypto/ssh"
)

type Payload struct {
	Str string
}

func main() {

	config := &ssh.ServerConfig{
		NoClientAuth: true,
	}

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key (./id_rsa)")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key")
	}

	config.AddHostKey(private)

	listener, err := net.Listen("tcp", "0.0.0.0:2022")
	if err != nil {
		log.Fatal("Failed to listen on 2022")
	}

	log.Print("Listening on 2022...")
	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Printf("Failed to handshake (%s)", err)
			continue
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		go handleRequests(reqs)
		go handleGitCommands(chans)

	}
}

func handleRequests(reqs <-chan *ssh.Request) {
	for req := range reqs {
		log.Printf("recieved out-of-band request: %+v", req)
	}
}

func handleGitCommands(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		channel, requets, _ := newChannel.Accept()
		go func(in <-chan *ssh.Request) {
			for req := range in {
				fmt.Printf("declining %s request...\n", req.Type)
				switch req.Type {
				case "exec":

					payload := Payload{}
					ssh.Unmarshal(req.Payload, &payload)

					parts := strings.Fields(payload.Str)
					fmt.Printf("run command: %s \n", payload.Str)
					command := string(parts[0])
					cmd := exec.Command("")

					switch command {
					case "git-upload-pack":
						cmd = exec.Command("/usr/bin/git-upload-pack", "/Users/stephenzhen/Projects/test.git")
					case "git-receive-pack":
						cmd = exec.Command("/usr/bin/git-receive-pack", "/Users/stephenzhen/Projects/test")
					case "git-upload-archive":
						cmd = exec.Command("/usr/bin/git-upload-archive", "/Users/stephenzhen/Projects/test.git")
					}
					out, _ := cmd.StdoutPipe()
					input, _ := cmd.StdinPipe()
					cmd.Stderr = os.Stderr
					go io.Copy(channel, out)
					go io.Copy(input, channel)
					cmd.Start()
					cmd.Wait()
					channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
					channel.Close()
				case "env":
					payload := Payload{}
					ssh.Unmarshal(req.Payload, &payload)
					fmt.Printf("run command: %s \n", payload.Str)
				default:
					channel.Close()
				}
			}
		}(requets)
	}
}

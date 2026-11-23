// Command debug-relay lets Claude look into a live playback-engine session.
// A game opened on a dev server connects to it by itself and records what
// happens from its start, so a session that got stuck can be read as it
// stands, without the reload that would destroy it.
//
// The relay listens on a fixed port of this machine, so only one runs, and
// holds one tab at a time; another tab is told it is busy and takes over
// once the first goes. Claude asks over HTTP:
//
//	curl localhost:7199/tab                        the tab's game, session and start
//	curl localhost:7199/snapshot                   its state and recent history
//	curl --data-binary 'playback.state.phases' localhost:7199/eval
//
// An expression runs in the tab, `await` included, with `playback`,
// `recorder` and `snapshot` in scope; its result comes back as JSON.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"
)

// Port is where every page looks for the relay, outside the dev stack's ports.
const Port = 7199

func main() {
	timeout := flag.Duration("timeout", 10*time.Second, "how long a tab may take to answer")
	flag.Parse()

	addr := fmt.Sprintf("127.0.0.1:%d", Port)
	listener, err := net.Listen("tcp", addr)
	if errors.Is(err, syscall.EADDRINUSE) {
		fmt.Fprintf(os.Stderr, "port %d is taken: a relay is running already\n", Port)
		os.Exit(1)
	}
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("debug relay on http://localhost:%d: waiting for a game on a dev server", Port)
	log.Fatal(http.Serve(listener, New(*timeout).Handler()))
}

package core

import (
	"testing"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// A host's callback is on the public internet: the platform never posts to
// its own machine or its network.
func TestTheHostCallbackReachesOnlyThePublicInternet(t *testing.T) {
	if schemafapi.DevMode() {
		t.Skip("development lets a callback run on the own machine")
	}
	for _, address := range []string{"127.0.0.1:443", "[::1]:443", "10.0.0.7:443", "192.168.1.2:80", "169.254.169.254:80", "0.0.0.0:443"} {
		if publicOnly("tcp", address, nil) == nil {
			t.Errorf("%s: reached", address)
		}
	}
	if err := publicOnly("tcp", "93.184.216.34:443", nil); err != nil {
		t.Errorf("a public address: %v", err)
	}
}

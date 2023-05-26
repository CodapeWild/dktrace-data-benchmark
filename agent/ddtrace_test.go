package agent

import (
	"testing"
)

func TestDDAgent(t *testing.T) {
	svr := NewDDAgent()
	svr.Start("127.0.0.1:9000")
	select {}
}

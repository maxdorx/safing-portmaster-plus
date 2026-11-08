package firewall

import (
	"context"
	"testing"

	"github.com/safing/portmaster/service/network"
)

func TestSPNTunnelingUnavailable(t *testing.T) {
	t.Parallel()

	// Even an old profile must not cause profile access, network lookup, or
	// fail-closed SPN readiness checks in this build.
	conn := &network.Connection{Verdict: network.VerdictAccept}
	checkTunneling(context.Background(), conn)
	if conn.Verdict != network.VerdictAccept || conn.Tunneled {
		t.Fatal("SPN altered a locally accepted connection")
	}
	err := requestTunneling(context.Background(), conn)
	if err == nil {
		t.Fatal("SPN tunnel request succeeded")
	}
}

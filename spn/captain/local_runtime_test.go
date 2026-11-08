package captain

import (
	"testing"

	"github.com/safing/portmaster/base/api"
	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/spn/conf"
)

func TestIntegratedCaptainIsDormant(t *testing.T) {
	conf.EnableIntegration(true)
	defer conf.EnableIntegration(false)
	// An absent instance is intentional: dormant construction must not access
	// SPN dependencies such as Patrol, updates, or the network environment.
	captain, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if captain.EventSPNConnected == nil || captain.HookSPNConnecting == nil || captain.States() == nil {
		t.Fatal("local status and event hooks are missing")
	}
	if captain.publicIdentityUpdater != nil || captain.statusUpdater != nil {
		t.Fatal("SPN workers initialized")
	}
	if _, err := api.GetEndpointByPath("spn/reinit"); err == nil {
		t.Error("SPN activation endpoint registered")
	}
	if _, err := config.GetOption("spn/enable"); err == nil {
		t.Error("SPN activation setting registered")
	}
	if err := captain.Start(); err != nil {
		t.Fatal(err)
	}
	if err := captain.Stop(); err != nil {
		t.Fatal(err)
	}
	if GetSPNStatus().Status != StatusDisabled {
		t.Error("SPN status is not disabled")
	}
}

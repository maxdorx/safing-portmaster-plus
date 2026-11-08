package captain

import (
	"testing"

	"github.com/safing/portmaster/base/api"
	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/spn/conf"
)

func TestIntegratedCaptainIsDormant(t *testing.T) { //nolint:paralleltest // constructs the singleton Captain module.
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
	_, err = api.GetEndpointByPath("spn/reinit")
	if err == nil {
		t.Error("SPN activation endpoint registered")
	}
	_, err = config.GetOption("spn/enable")
	if err == nil {
		t.Error("SPN activation setting registered")
	}
	err = captain.Start()
	if err != nil {
		t.Fatal(err)
	}
	err = captain.Stop()
	if err != nil {
		t.Fatal(err)
	}
	if GetSPNStatus().Status != StatusDisabled {
		t.Error("SPN status is not disabled")
	}
}

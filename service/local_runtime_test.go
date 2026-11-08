package service

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

	"github.com/safing/portmaster/base/api"
	"github.com/safing/portmaster/base/config"
	"github.com/safing/portmaster/spn/conf"
)

func TestLocalServiceHasNoSPNRuntime(t *testing.T) { //nolint:paralleltest // construction initializes singleton modules.
	if runtime.GOOS == "windows" {
		t.Skip("service data-directory construction requires an administrative Windows token")
	}
	// Production modules intentionally allow one instance per process. Keep this
	// full-construction regression isolated from TestDebug and other unit tests.
	if os.Getenv("PORTMASTER_SERVICE_CONSTRUCTION_TEST") != "1" {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(executable, "-test.run=^TestLocalServiceHasNoSPNRuntime$")
		cmd.Env = append(os.Environ(), "PORTMASTER_SERVICE_CONSTRUCTION_TEST=1")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("isolated service construction: %v\n%s", err, output)
		}
		return
	}
	// Embedded callers receive the same local-only service as the core CLI.
	conf.EnableIntegration(false)
	conf.EnableClient(true)
	dir := t.TempDir()
	svc, err := New(&ServiceConfig{BinDir: dir, DataDir: dir, LogToStdout: true})
	if err != nil {
		t.Fatal(err)
	}
	if conf.Client() || !conf.Integrated() {
		t.Fatal("local service enabled SPN client mode")
	}
	if len(svc.SPNGroup().Modules()) != 0 {
		t.Fatal("SPN network modules are available to start")
	}
	if svc.Captain() == nil || svc.GetEventSPNConnected() == nil || svc.GetHookSPNConnecting() == nil {
		t.Fatal("local resolver/interoperability event dependencies are missing")
	}
	if svc.Broadcasts() != nil {
		t.Fatal("promotional broadcasts module was constructed")
	}
	if _, err := config.GetOption("spn/enable"); err == nil {
		t.Error("SPN activation setting registered")
	}
	for _, path := range []string{"spn/reinit", "spn/account/login", "spn/account/logout", "broadcasts/simulate"} {
		if _, err := api.GetEndpointByPath(path); err == nil {
			t.Errorf("unavailable runtime endpoint registered: %s", path)
		}
	}
	// These are invoked by status/sleep/pause callers before SPN would start.
	svc.SetSleep(true)
	_ = svc.GetStates()
	if stopped, err := svc.SPNGroup().IsStopped(); err != nil || !stopped {
		t.Errorf("SPN group not stopped: %v, %v", stopped, err)
	}
}

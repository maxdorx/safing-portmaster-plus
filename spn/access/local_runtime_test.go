package access

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/safing/portmaster/base/api"
	"github.com/safing/portmaster/spn/access/account"
	"github.com/safing/portmaster/spn/conf"
)

// With integration enabled even a direct caller cannot use account credentials,
// mutate stored account state, or trigger a token request.
func TestIntegratedAccountOperationsUnavailable(t *testing.T) {
	previousIntegration := conf.Integrated()
	conf.EnableIntegration(true)
	defer conf.EnableIntegration(previousIntegration)
	previousClient := accountClient
	defer func() { accountClient = previousClient }()
	requests := 0
	accountClient = &http.Client{Transport: localOnlyTransport(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, errors.New("unexpected account network request")
	})}
	if _, _, err := Login("stored-user", "password"); err == nil {
		t.Error("Login succeeded")
	}
	if err := Logout(false, true); err == nil {
		t.Error("Logout allowed stored credentials to be purged")
	}
	if _, _, err := UpdateUser(); err == nil {
		t.Error("UpdateUser succeeded")
	}
	if err := UpdateTokens(); err == nil {
		t.Error("UpdateTokens succeeded")
	}
	if err := UpdateAccount(nil); err == nil {
		t.Error("UpdateAccount succeeded")
	}
	if _, err := makeClientRequest(nil); err == nil {
		t.Error("account transport was available")
	}
	if requests != 0 {
		t.Errorf("made %d account network requests", requests)
	}
}

func TestIntegratedAccessRetainsCredentials(t *testing.T) {
	if module.updateAccountWorkerMgr != nil {
		t.Fatal("integrated runtime created an account worker")
	}
	for _, path := range []string{"spn/account/login", "spn/account/logout"} {
		if _, err := api.GetEndpointByPath(path); err == nil {
			t.Errorf("mutation endpoint registered: %s", path)
		}
	}
	if _, err := api.GetEndpointByPath("spn/account/user/profile"); err != nil {
		t.Error("local feature plan endpoint unavailable")
	}
	credential := &AuthTokenRecord{Token: &account.AuthToken{Device: "stored-device", Token: "stored-token"}}
	if err := credential.Save(); err != nil {
		t.Fatal(err)
	}
	if err := module.Stop(); err != nil {
		t.Fatal(err)
	}
	clearUserCaches()
	saved, err := GetAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Token.Device != "stored-device" || saved.Token.Token != "stored-token" {
		t.Fatal("stored credentials changed")
	}
}

func TestLegacyEnabledSPNStaysStoppedOnConfigChange(t *testing.T) {
	group := module.instance.SPNGroup()
	if stopped, err := group.IsStopped(); err != nil || !stopped {
		t.Fatalf("legacy setting activated SPN at startup: %v", err)
	}
	module.instance.Config().EventConfigChange.Submit(struct{}{})
	if !module.instance.Config().Manager().WaitForWorkers(5*time.Second) || !module.Manager().WaitForWorkers(5*time.Second) {
		t.Fatal("configuration workers did not finish")
	}
	if stopped, err := group.IsStopped(); err != nil || !stopped {
		t.Fatalf("configuration change activated SPN: %v", err)
	}
}

type localOnlyTransport func(*http.Request) (*http.Response, error)

func (f localOnlyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

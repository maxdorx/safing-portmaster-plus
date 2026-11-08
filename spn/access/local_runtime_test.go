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

var errUnexpectedAccountRequest = errors.New("unexpected account network request")

// With integration enabled even a direct caller cannot use account credentials,
// mutate stored account state, or trigger a token request.
func TestIntegratedAccountOperationsUnavailable(t *testing.T) { //nolint:paralleltest // uses shared account module and configuration.
	previousIntegration := conf.Integrated()
	conf.EnableIntegration(true)
	defer conf.EnableIntegration(previousIntegration)
	previousClient := accountClient
	defer func() { accountClient = previousClient }()
	requests := 0
	accountClient = &http.Client{Transport: localOnlyTransport(func(*http.Request) (*http.Response, error) {
		requests++
		return nil, errUnexpectedAccountRequest
	})}
	_, _, err := Login("stored-user", "password")
	if err == nil {
		t.Error("Login succeeded")
	}
	err = Logout(false, true)
	if err == nil {
		t.Error("Logout allowed stored credentials to be purged")
	}
	_, _, err = UpdateUser()
	if err == nil {
		t.Error("UpdateUser succeeded")
	}
	err = UpdateTokens()
	if err == nil {
		t.Error("UpdateTokens succeeded")
	}
	err = UpdateAccount(nil)
	if err == nil {
		t.Error("UpdateAccount succeeded")
	}
	response, requestErr := makeClientRequest(nil)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if requestErr == nil {
		t.Error("account transport was available")
	}
	if requests != 0 {
		t.Errorf("made %d account network requests", requests)
	}
}

func TestIntegratedAccessRetainsCredentials(t *testing.T) { //nolint:paralleltest // uses shared account module and configuration.
	if module.updateAccountWorkerMgr != nil {
		t.Fatal("integrated runtime created an account worker")
	}
	for _, path := range []string{"spn/account/login", "spn/account/logout"} {
		_, err := api.GetEndpointByPath(path)
		if err == nil {
			t.Errorf("mutation endpoint registered: %s", path)
		}
	}
	_, err := api.GetEndpointByPath("spn/account/user/profile")
	if err != nil {
		t.Error("local feature plan endpoint unavailable")
	}
	credential := &AuthTokenRecord{Token: &account.AuthToken{Device: localTestStoredDevice, Token: "stored-token"}}
	err = credential.Save()
	if err != nil {
		t.Fatal(err)
	}
	err = module.Stop()
	if err != nil {
		t.Fatal(err)
	}
	clearUserCaches()
	saved, err := GetAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Token.Device != localTestStoredDevice || saved.Token.Token != "stored-token" {
		t.Fatal("stored credentials changed")
	}
}

func TestLegacyEnabledSPNStaysStoppedOnConfigChange(t *testing.T) { //nolint:paralleltest // uses shared account module and configuration.
	group := module.instance.SPNGroup()
	stopped, err := group.IsStopped()
	if err != nil || !stopped {
		t.Fatalf("legacy setting activated SPN at startup: %v", err)
	}
	module.instance.Config().EventConfigChange.Submit(struct{}{})
	if !module.instance.Config().Manager().WaitForWorkers(5*time.Second) || !module.Manager().WaitForWorkers(5*time.Second) {
		t.Fatal("configuration workers did not finish")
	}
	stopped, err = group.IsStopped()
	if err != nil || !stopped {
		t.Fatalf("configuration change activated SPN: %v", err)
	}
}

type localOnlyTransport func(*http.Request) (*http.Response, error)

func (f localOnlyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

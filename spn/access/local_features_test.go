package access

import (
	"slices"
	"testing"
	"time"

	"github.com/safing/portmaster/spn/access/account"
)

const localTestStoredDevice = "stored-device"

// TestApplyLocalFeaturesReleasesLock guards against the regression that took
// the firewall down: the first version held the record lock across Save, which
// re-locks the record via db.Put. sync.Mutex is not reentrant, so it
// self-deadlocked inside module start, leaving the kernel extension holding
// every packet with no userspace left to issue verdicts.
func TestApplyLocalFeaturesReleasesLock(t *testing.T) {
	t.Parallel()

	user := &UserRecord{}
	if !applyLocalFeatures(user) {
		t.Fatal("applyLocalFeatures reported no change on an empty record")
	}

	// Save runs next and locks the record again. If the lock is still held
	// here, that call deadlocks.
	if !user.TryLock() {
		t.Fatal("record still locked after applyLocalFeatures: Save would deadlock")
	}
	user.Unlock()
}

func TestLocalPlanReplacesStoredSubscriptionFeatures(t *testing.T) {
	t.Parallel()
	endsAt := time.Now().Add(time.Hour)
	user := &UserRecord{User: &account.User{
		State:        account.UserStateApproved,
		Username:     "stored-user",
		Device:       &account.Device{ID: localTestStoredDevice},
		Subscription: &account.Subscription{EndsAt: &endsAt},
		CurrentPlan:  &account.Plan{Name: "Unlimited", FeatureIDs: []account.FeatureID{account.FeatureSPN, account.FeatureSafingSupport}},
		View:         &account.View{ShowLoginButton: true, ShowAccountButton: true},
	}}
	if !applyLocalFeatures(user) {
		t.Fatal("stored subscription did not receive local plan")
	}
	if user.CurrentPlan.Name != "Portmaster Plus" {
		t.Errorf("plan = %q", user.CurrentPlan.Name)
	}
	if !slices.Equal(user.CurrentPlan.FeatureIDs, locallyEnabledFeatures) {
		t.Errorf("plan features = %v", user.CurrentPlan.FeatureIDs)
	}
	if user.MayUseSPN() || user.MayUseTheSPN() || user.MayUse(account.FeatureSPN) {
		t.Error("stored SPN subscription still grants SPN access")
	}
	if user.View.ShowLoginButton || user.View.ShowAccountButton {
		t.Error("local plan exposes account actions")
	}
	if user.Username != "stored-user" || user.Device.ID != localTestStoredDevice || user.Subscription.EndsAt != &endsAt {
		t.Error("stored credentials or subscription were discarded")
	}
	if applyLocalFeatures(user) {
		t.Error("second apply should be a no-op")
	}
}

func TestApplyLocalFeaturesContent(t *testing.T) {
	t.Parallel()

	user := &UserRecord{}
	applyLocalFeatures(user)

	// The UI derives every feature gate from CurrentPlan.FeatureIDs, so the
	// stored record - not just MayUse - has to carry them.
	if user.CurrentPlan == nil {
		t.Fatal("no current plan set")
	}
	for _, featureID := range locallyEnabledFeatures {
		if !slices.Contains(user.CurrentPlan.FeatureIDs, featureID) {
			t.Errorf("feature %q missing from plan", featureID)
		}
		if !user.MayUse(featureID) {
			t.Errorf("MayUse(%q) = false, want true", featureID)
		}
	}

	// Server-backed access must stay gated, and we must not look logged in.
	if user.MayUse(account.FeatureSPN) {
		t.Error("MayUse(spn) = true, want false: SPN cannot be enabled locally")
	}
	if user.IsLoggedIn() {
		t.Error("IsLoggedIn() = true, want false")
	}

	// The view must be preset, or Save's UpdateView fallback resets the plan
	// to "Free" and drops the feature IDs.
	if user.View == nil {
		t.Error("view not set: UpdateView would reset the plan to Free")
	}

	// Reapplying must be a no-op rather than duplicating entries.
	before := len(user.CurrentPlan.FeatureIDs)
	if applyLocalFeatures(user) {
		t.Error("second apply reported a change, want no-op")
	}
	if got := len(user.CurrentPlan.FeatureIDs); got != before {
		t.Errorf("feature count changed on reapply: got %d, want %d", got, before)
	}
}

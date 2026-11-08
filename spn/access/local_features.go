package access

import (
	"errors"
	"reflect"
	"slices"

	"github.com/safing/portmaster/spn/access/account"
)

// localPlanName is the plan name reported when this build enables features
// without an account.
const localPlanName = "Portmaster Plus"

// locallyEnabledFeatures lists the features that this build makes available
// without an account subscription.
//
// Every feature listed here is implemented entirely on this device: it reads
// and writes local state only and never contacts Safing infrastructure.
// Restricting them is a licensing decision rather than a technical one, so a
// self-compiled build can enable them without side effects.
//
// Features that depend on Safing's servers are deliberately absent. The SPN in
// particular cannot be enabled this way: its access tokens are blind-signed by
// the account server and verified by every Hub before an operation is granted
// (see op_auth.go and zones.go), so listing account.FeatureSPN here would only
// produce failing connection attempts and error notifications.
var locallyEnabledFeatures = []account.FeatureID{
	// Network History: persists connections to the local SQLite history
	// database. Also requires the "history/enable" setting.
	account.FeatureHistory,

	// Bandwidth Visibility: records per-connection byte counters.
	account.FeatureBWVis,
}

// isLocallyEnabled reports whether the given feature is enabled by this build
// independently of the account subscription.
func isLocallyEnabled(featureID account.FeatureID) bool {
	return slices.Contains(locallyEnabledFeatures, featureID)
}

// applyLocalFeatures adds the locally enabled features to the user's plan and
// reports whether anything changed.
//
// This holds the record lock and must therefore not perform any I/O: saving
// goes through db.Put, which locks the record again, and sync.Mutex is not
// reentrant. Keeping the mutation separate from the save is what stops that
// deadlock from reappearing.
func applyLocalFeatures(user *UserRecord) (changed bool) {
	user.Lock()
	defer user.Unlock()

	if user.User == nil {
		user.User = &account.User{State: account.UserStateLoggedOut}
	}
	plan := &account.Plan{Name: localPlanName, FeatureIDs: slices.Clone(locallyEnabledFeatures)}
	if !reflect.DeepEqual(user.CurrentPlan, plan) {
		user.CurrentPlan = plan
		changed = true
	}
	// Preserve account credentials and subscription metadata, but always expose
	// the local plan and no account actions. Save must not run UpdateView here.
	view := &account.View{Message: "Portmaster Plus: local features are enabled."}
	if !reflect.DeepEqual(user.View, view) {
		user.View = view
		changed = true
	}
	return changed
}

// ensureLocalFeaturePlan makes sure the stored user record advertises the
// locally enabled features.
func ensureLocalFeaturePlan() error {
	user, err := GetUser()
	switch {
	case err == nil:
		// Keep stored identity and credentials while publishing the local plan.
	case errors.Is(err, ErrNotLoggedIn):
		user = &UserRecord{}
	default:
		return err
	}

	if !applyLocalFeatures(user) {
		return nil
	}

	// Save with the record unlocked; see applyLocalFeatures.
	return user.Save()
}

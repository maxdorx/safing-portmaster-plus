package access

import (
	"errors"
	"slices"

	"github.com/safing/portmaster/spn/access/account"
)

// localPlanName is the plan name reported when this build enables features
// without an account.
const localPlanName = "Local Build"

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

// ensureLocalFeaturePlan makes sure the stored user record advertises the
// locally enabled features.
//
// The user interface never consults MayUse. It watches the user record in the
// database (core:spn/account/user) and derives every feature gate from
// CurrentPlan.FeatureIDs, so a backend-only change leaves the interface locked
// even though the feature works. Publishing the same allowlist through the
// record keeps both sides consistent.
//
// The record stays in the logged out state: IsLoggedIn keeps reporting false,
// no request is ever made to Safing's servers, and logging in for real still
// replaces the record wholesale.
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
	if user.CurrentPlan == nil {
		user.CurrentPlan = &account.Plan{Name: localPlanName}
	}

	// Add missing local features, leaving a real subscription intact.
	for _, featureID := range locallyEnabledFeatures {
		if !slices.Contains(user.CurrentPlan.FeatureIDs, featureID) {
			user.CurrentPlan.FeatureIDs = append(user.CurrentPlan.FeatureIDs, featureID)
			changed = true
		}
	}
	if !changed {
		return false
	}

	// Set the view explicitly. Save falls back to UpdateView, which resets the
	// plan to "Free" - dropping the feature IDs - when there is no subscription.
	if user.View == nil {
		user.View = &account.View{
			Message:           "Local build: device-local features are enabled without an account.",
			ShowAccountButton: true,
			ShowLoginButton:   true,
		}
	}

	return true
}

// ensureLocalFeaturePlan makes sure the stored user record advertises the
// locally enabled features.
func ensureLocalFeaturePlan() error {
	user, err := GetUser()
	switch {
	case err == nil:
		// A record already exists; only extend the plan it carries.
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

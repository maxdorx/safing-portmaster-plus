package access

import (
	"slices"

	"github.com/safing/portmaster/spn/access/account"
)

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

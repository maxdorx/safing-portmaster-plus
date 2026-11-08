package firewall

import (
	"context"

	"github.com/safing/portmaster/base/log"
	"github.com/safing/portmaster/service/intel"
	"github.com/safing/portmaster/service/network"
	"github.com/safing/portmaster/service/process"
	"github.com/safing/portmaster/service/profile"
	"github.com/safing/portmaster/service/profile/endpoints"
	"github.com/safing/portmaster/service/resolver"
	"github.com/safing/portmaster/spn/access"
	"github.com/safing/portmaster/spn/captain"
	"github.com/safing/portmaster/spn/navigator"
)

// checkTunneling ignores legacy SPN settings and profiles in this local build.
func checkTunneling(_ context.Context, _ *network.Connection) {}

func requestTunneling(_ context.Context, _ *network.Connection) error {
	return access.ErrSPNUnavailable
}

func init() {
	navigator.DeriveTunnelOptions = func(lp *profile.LayeredProfile, destination *intel.Entity, connEncrypted bool) *navigator.Options {
		return DeriveTunnelOptions(lp, nil, destination, connEncrypted)
	}
}

// DeriveTunnelOptions derives and returns the tunnel options from the connection and profile.
func DeriveTunnelOptions(lp *profile.LayeredProfile, proc *process.Process, destination *intel.Entity, connEncrypted bool) *navigator.Options {
	// Set options.
	tunnelOpts := &navigator.Options{
		Transit: &navigator.TransitHubOptions{
			HubPolicies: lp.StackedTransitHubPolicies(),
		},
		Destination: &navigator.DestinationHubOptions{
			HubPolicies:        lp.StackedExitHubPolicies(),
			CheckHubPolicyWith: destination,
		},
		RoutingProfile: lp.SPNRoutingAlgorithm(),
	}
	if !connEncrypted {
		tunnelOpts.Destination.Regard = tunnelOpts.Destination.Regard.Add(navigator.StateTrusted)
		// TODO: Add this when all Hubs are on v0.6.21+
		// tunnelOpts.Destination.Regard = tunnelOpts.Destination.Regard.Add(navigator.StateAllowUnencrypted)
	}

	// Add required verified owners if community nodes should not be used.
	if !useCommunityNodes() {
		tunnelOpts.Transit.RequireVerifiedOwners = captain.NonCommunityVerifiedOwners
		tunnelOpts.Destination.RequireVerifiedOwners = captain.NonCommunityVerifiedOwners
	}

	// Get routing profile for checking for upgrades.
	routingProfile := navigator.GetRoutingProfile(tunnelOpts.RoutingProfile)

	// If we have any exit hub policies, we must be able to hop in order to follow the policy.
	// Switch to single-hop routing to allow for routing with hub selection.
	if routingProfile.MaxHops <= 1 && navigator.HubPoliciesAreSet(tunnelOpts.Destination.HubPolicies) {
		tunnelOpts.RoutingProfile = navigator.RoutingProfileSingleHopID
	}

	// If the current home node is not trusted, then upgrade at least to two hops.
	if routingProfile.MinHops < 2 && navigator.Main != nil {
		homeNode, _ := navigator.Main.GetHome()
		if homeNode != nil && !homeNode.State.Has(navigator.StateTrusted) {
			tunnelOpts.RoutingProfile = navigator.RoutingProfileDoubleHopID
		}
	}

	// Special handling for the internal DNS resolver.
	if proc != nil && proc.Pid == ownPID && resolver.IsResolverAddress(destination.IP, destination.Port) {
		dnsExitHubPolicy, err := captain.GetDNSExitHubPolicy()
		if err != nil {
			log.Errorf("firewall: failed to get dns exit hub policy: %s", err)
		}

		if err == nil && dnsExitHubPolicy.IsSet() {
			// Apply the dns exit hub policy, if set.
			tunnelOpts.Destination.HubPolicies = []endpoints.Endpoints{dnsExitHubPolicy}
			// Use the routing algorithm from the profile, as the home profile won't work with the policy.
			tunnelOpts.RoutingProfile = lp.SPNRoutingAlgorithm()
			// Raise the routing algorithm at least to single-hop.
			if tunnelOpts.RoutingProfile == navigator.RoutingProfileHomeID {
				tunnelOpts.RoutingProfile = navigator.RoutingProfileSingleHopID
			}
		} else {
			// Disable any policies for the internal DNS resolver.
			tunnelOpts.Destination.HubPolicies = nil
			// Always use the home routing profile for the internal DNS resolver.
			tunnelOpts.RoutingProfile = navigator.RoutingProfileHomeID
		}
	}

	return tunnelOpts
}

// Package resolver translates custom fields and legacy tags into effective settings.
package resolver

import (
	"slices"

	"peg.nu/nx/model"
	"peg.nu/nx/tagparser"
)

// forwardState tracks whether suppression came from a cleared zone rather than
// an explicit flag. A child zone can undo legacy clearing, but not an explicit
// prefix-wide disable flag. This state never escapes the resolver.
type forwardState struct {
	disabledByEmptyZone bool
}

func overlay(target *model.Configuration, source model.CustomFields, forward *forwardState) {
	if source.DNSEnabled != nil {
		target.DNSEnabled = *source.DNSEnabled
	}
	if source.DNSForwardZone != nil {
		if *source.DNSForwardZone == "" {
			// Preserve the inherited zone for PTR targets.
			target.DNSForwardDisabled = true
			forward.disabledByEmptyZone = true
		} else {
			target.DNSForwardZone = *source.DNSForwardZone
			if forward.disabledByEmptyZone {
				target.DNSForwardDisabled = false
			}
			forward.disabledByEmptyZone = false
		}
	}
	if source.DNSForwardDisabled != nil {
		target.DNSForwardDisabled = *source.DNSForwardDisabled
		forward.disabledByEmptyZone = false
	}
	if source.DNSReverseZone != nil {
		target.DNSReverseZone = *source.DNSReverseZone
	}
	if source.DNSCNames != nil {
		target.DNSCNames = slices.Clone(*source.DNSCNames)
	}
	if source.IPLEnabled != nil {
		target.IPLEnabled = *source.IPLEnabled
	}
	if source.IPLists != nil {
		target.IPLists = slices.Clone(*source.IPLists)
	}
}

func local(target *model.Configuration, tags []model.Tag, fields model.CustomFields, forward *forwardState) {
	var legacy model.CustomFields
	tagparser.ParseTags(&legacy, tags, nil)
	// A supplied custom zone replaces the legacy zone and its clearing effect.
	if fields.DNSForwardZone != nil {
		legacy.DNSForwardZone = nil
	}
	overlay(target, legacy, forward)
	overlay(target, fields, forward)
}

func ResolvePrefix(prefix *model.IPAMPrefix) {
	prefix.Config = model.Configuration{}
	local(&prefix.Config, prefix.Tags, prefix.CustomFields, &forwardState{})
}

// ResolveAddress applies address custom fields > address tags > prefix settings.
// Prefix selection remains a separate gate: an address cannot enable its prefix.
func ResolveAddress(address *model.IPAddress) {
	address.Config = model.Configuration{}
	forward := &forwardState{}
	if address.Prefix != nil {
		// Resolve both layers together to retain the provenance of legacy clearing.
		local(&address.Config, address.Prefix.Tags, address.Prefix.CustomFields, forward)
	}
	local(&address.Config, address.Tags, address.CustomFields, forward)
}

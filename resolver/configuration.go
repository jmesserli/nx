// Package resolver applies custom-field overrides to inherited prefix settings.
package resolver

import (
	"slices"

	"peg.nu/nx/model"
)

func overlay(target *model.Configuration, source model.CustomFields) {
	if source.DNSEnabled != nil {
		target.DNSEnabled = *source.DNSEnabled
	}
	if source.DNSForwardZone != nil {
		target.DNSForwardZone = *source.DNSForwardZone
	}
	if source.DNSForwardDisabled != nil {
		target.DNSForwardDisabled = *source.DNSForwardDisabled
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

func ResolvePrefix(prefix *model.IPAMPrefix) {
	prefix.Config = model.Configuration{}
	overlay(&prefix.Config, prefix.CustomFields)
}

// ResolveAddress applies address custom fields over prefix custom fields.
// Prefix selection remains a separate gate: an address cannot enable its prefix.
func ResolveAddress(address *model.IPAddress) {
	address.Config = model.Configuration{}
	if address.Prefix != nil {
		overlay(&address.Config, address.Prefix.CustomFields)
	}
	overlay(&address.Config, address.CustomFields)
}

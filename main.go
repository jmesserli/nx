package main

import (
	"fmt"

	"github.com/jmesserli/netbox-to-bind/bind"
	"github.com/jmesserli/netbox-to-bind/netbox"
)

func main() {
	prefixes := netbox.GetIPAMPrefixes()

	addressList := []netbox.IPAddress{}

	for _, prefix := range prefixes {
		if prefix.GenOptions.Enabled {
			fmt.Printf("%v:\n", prefix.Prefix)

			addresses := netbox.GetIPAddressesByPrefix(&prefix)
			addressList = append(addressList, addresses...)
		}
	}

	bind.GenerateZones(addressList, bind.SOAInfo{
		BindDefaultRRTTL: 86400,
		Expire:           86400,
		Refresh:          1800,
		Retry:            1800,
		TTL:              600,

		DottedMailResponsible: "postmaster.peg.nu.",
		NameserverFQDN:        "vm-ns-1.bue39.pegnu.net.",
	})
}

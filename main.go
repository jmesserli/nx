package main

import (
	"fmt"

	"github.com/jmesserli/netbox-to-bind/bind"

	"github.com/jmesserli/netbox-to-bind/netbox"
)

func main() {
	prefixes := netbox.GetIPAMPrefixes()

	for _, prefix := range prefixes {
		if prefix.GenOptions.Enabled {
			fmt.Printf("%v:\n", prefix.Prefix)

			addresses := netbox.GetIPAddressesByPrefix(&prefix)
			for _, address := range addresses {

				if address.GenOptions.Enabled {
					fmt.Printf("		%v: %v\n", address.Address, address.Name)
				}
			}

			bind.GenerateZones(addresses, bind.SOAInfo{})
		}
	}
}

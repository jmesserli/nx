package main

import (
	"log"
	"os"

	"github.com/jmesserli/netbox-to-bind/bind"
	"github.com/jmesserli/netbox-to-bind/netbox"
)

var logger = log.New(os.Stdout, "[main] ", log.LstdFlags)

func main() {
	logger.Println("Loading prefixes")
	prefixes := netbox.GetIPAMPrefixes()

	addressList := []netbox.IPAddress{}

	logger.Println("Loading ip addresses of enabled prefixes")
	for _, prefix := range prefixes {
		if prefix.GenOptions.Enabled {
			addresses := netbox.GetIPAddressesByPrefix(&prefix)
			addressList = append(addressList, addresses...)
		}
	}

	logger.Println("Generating zone files")
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

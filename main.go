package main

import (
	"log"
	"os"
	"time"

	"github.com/jmesserli/netbox-to-bind/config"

	"github.com/jmesserli/netbox-to-bind/bind"
	"github.com/jmesserli/netbox-to-bind/netbox"
)

var logger = log.New(os.Stdout, "[main] ", log.LstdFlags)

func main() {
	conf := config.ReadConfig("./config.json")

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
	generatedZones := bind.GenerateZones(addressList, bind.SOAInfo{
		BindDefaultRRTTL: int(2 * time.Minute / time.Second),
		Expire:           int(48 * time.Hour / time.Second),
		Refresh:          int(15 * time.Minute / time.Second),
		Retry:            int(15 * time.Minute / time.Second),
		TTL:              int(10 * time.Minute / time.Second),

		DottedMailResponsible: "postmaster.peg.nu.",
		NameserverFQDN:        "vm-ns-1.bue39.pegnu.net.",
	})
	bind.GenerateConfigs(generatedZones, conf)
}

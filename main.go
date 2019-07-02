package main

import (
	"log"
	"os"
	"time"

	"github.com/jmesserli/nx/config"

	"github.com/jmesserli/nx/bind"
	"github.com/jmesserli/nx/netbox"
)

var logger = log.New(os.Stdout, "[main] ", log.LstdFlags)

func main() {
	conf := config.ReadConfig("./config.json")

	nc := netbox.New(conf)
	logger.Println("Loading prefixes")
	prefixes := nc.GetIPAMPrefixes()

	addressList := []netbox.IPAddress{}

	logger.Println("Loading ip addresses of enabled prefixes")
	for _, prefix := range prefixes {
		if prefix.GenOptions.Enabled {
			addresses := nc.GetIPAddressesByPrefix(&prefix)
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

		DottedMailResponsible: "unknown\\.admin.local",
		NameserverFQDN:        "unknown-nameserver.local.",
	}, conf)
	bind.GenerateConfigs(generatedZones, conf)
}

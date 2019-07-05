package main

import (
	"log"
	"os"
	"time"

	"github.com/jmesserli/nx/config"
	"github.com/jmesserli/nx/netbox"
	"github.com/jmesserli/nx/ns/dns"
)

var logger = log.New(os.Stdout, "[main] ", log.LstdFlags)

func main() {
	conf := config.ReadConfig("./config.json")

	nc := netbox.New(conf)
	logger.Println("Loading prefixes")
	prefixes := nc.GetIPAMPrefixes()

	dnsIps := []netbox.IPAddress{}
	wgIps := []netbox.IPAddress{}

	logger.Println("Loading ip addresses of enabled prefixes")
	for _, prefix := range prefixes {
		if !(prefix.EnOptions.DNSEnabled || prefix.EnOptions.WGEnabled) {
			continue
		}

		addresses := nc.GetIPAddressesByPrefix(prefix)
		if prefix.EnOptions.DNSEnabled {
			dnsIps = append(dnsIps, addresses...)
		}
		if prefix.EnOptions.WGEnabled {
			wgIps = append(wgIps, addresses...)
		}
	}

	logger.Println("Generating dns zone files")
	generatedZones := dns.GenerateZones(dnsIps, dns.SOAInfo{
		BindDefaultRRTTL: int(2 * time.Minute / time.Second),
		Expire:           int(48 * time.Hour / time.Second),
		Refresh:          int(15 * time.Minute / time.Second),
		Retry:            int(15 * time.Minute / time.Second),
		TTL:              int(10 * time.Minute / time.Second),

		DottedMailResponsible: "unknown\\.admin.local",
		NameserverFQDN:        "unknown-nameserver.local.",
	}, conf)

	logger.Println("Generating BIND config files")
	dns.GenerateConfigs(generatedZones, conf)
}

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"peg.nu/nx/ns/ipl"
	"strings"
	"time"

	"peg.nu/nx/config"
	"peg.nu/nx/netbox"
	"peg.nu/nx/ns/dns"
	"peg.nu/nx/ns/wg"
)

var logger = log.New(os.Stdout, "[main] ", log.LstdFlags)

func main() {
	conf := config.ReadConfig("./config.json")

	nc := netbox.New(conf)
	logger.Println("Loading prefixes")
	prefixes := nc.GetIPAMPrefixes()

	var dnsIps, wgIps, iplIps []netbox.IPAddress

	logger.Println("Loading ip addresses of enabled prefixes")
	for _, prefix := range prefixes {
		if !(prefix.EnOptions.DNSEnabled || len(prefix.EnOptions.WGVpnName) > 0 || prefix.EnOptions.IPLEnabled) {
			logger.Println(fmt.Sprintf("Skipping prefix %s because no nx-features are enabled", prefix.Prefix))
			continue
		}

		logger.Println(fmt.Sprintf("Getting ip addresses in %s", prefix.Prefix))
		addresses := nc.GetIPAddressesByPrefix(prefix)
		if prefix.EnOptions.DNSEnabled {
			dnsIps = append(dnsIps, addresses...)
		}
		if len(prefix.EnOptions.WGVpnName) > 0 {
			wgIps = append(wgIps, addresses...)
		}
		if prefix.EnOptions.IPLEnabled {
			iplIps = append(iplIps, addresses...)
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
	}, &conf)

	logger.Println("Generating BIND config files")
	dns.GenerateConfigs(generatedZones, &conf)
	logger.Println("Generating Wireguard config files")
	wg.GenerateWgConfigs(wgIps, &conf)
	logger.Println("Generating IP lists")
	ipl.GenerateIPLists(iplIps, &conf)

	logger.Println("Writing updated files report")
	err := ioutil.WriteFile("generated/updated_files.txt", []byte(strings.Join(conf.UpdatedFiles, "\n")), os.ModePerm)
	if err != nil {
		logger.Fatal(err)
	}
}

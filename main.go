package main

import (
	"fmt"
	"log"
	"os"
	"peg.nu/nx/model"
	"peg.nu/nx/ns/ipl"
	"peg.nu/nx/util"
	"sort"
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
	if len(prefixes) == 0 {
		panic(fmt.Errorf("could not load prefixes: 0 prefixes loaded"))
	}

	var dnsIps, wgIps, iplIps []model.IPAddress

	prefixIPsList := loadPrefixes(prefixes, nc)
	sortPrefixList(prefixIPsList)
	generateAll(prefixIPsList, dnsIps, wgIps, iplIps, &conf)

	logger.Println("Writing updated files report")
	err := os.WriteFile("generated/updated_files.txt", []byte(strings.Join(conf.UpdatedFiles, "\n")), os.ModePerm)
	if err != nil {
		logger.Fatal(err)
	}
}

func loadPrefixes(prefixes []model.IPAMPrefix, nc netbox.Client) []prefixIPs {
	defer util.DurationSince(util.StartTracking("loadPrefixes"))

	logger.Println("Loading ip addresses of enabled prefixes")

	var enabledPrefixCount = 0
	var prefixIPsList []prefixIPs
	var prefixIPchan = make(chan prefixIPs)
	for _, prefix := range prefixes {
		if !(prefix.EnOptions.DNSEnabled || len(prefix.EnOptions.WGVpnName) > 0 || prefix.EnOptions.IPLEnabled) {
			//logger.Println(fmt.Sprintf("Skipping prefix %s because no nx-features are enabled", prefix.Prefix))
			continue
		}

		enabledPrefixCount++
		go getIPsForPrefix(nc, prefix, prefixIPchan)
	}

	for i := 0; i < enabledPrefixCount; i++ {
		prefixIPsList = append(prefixIPsList, <-prefixIPchan)
	}
	return prefixIPsList
}

func sortPrefixList(prefixIPsList []prefixIPs) {
	defer util.DurationSince(util.StartTracking("sortPrefixList"))

	sort.Slice(prefixIPsList, func(i, j int) bool {
		return util.CompareCIDRStrings(prefixIPsList[i].prefix.Prefix, prefixIPsList[j].prefix.Prefix)
	})

	for _, pip := range prefixIPsList {
		sort.Slice(pip.ips, util.IpAddressesLessFn(pip.ips))
	}
}

func generateAll(prefixIPsList []prefixIPs, dnsIps []model.IPAddress, wgIps []model.IPAddress, iplIps []model.IPAddress, conf *config.NXConfig) {
	defer util.DurationSince(util.StartTracking("generateAll"))

	for _, prefixIP := range prefixIPsList {
		if prefixIP.prefix.EnOptions.DNSEnabled {
			dnsIps = append(dnsIps, prefixIP.ips...)
		}
		if len(prefixIP.prefix.EnOptions.WGVpnName) > 0 {
			wgIps = append(wgIps, prefixIP.ips...)
		}
		if prefixIP.prefix.EnOptions.IPLEnabled {
			iplIps = append(iplIps, prefixIP.ips...)
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
	logger.Println("Generating Wireguard config files")
	wg.GenerateWgConfigs(wgIps, conf)
	logger.Println("Generating IP lists")
	ipl.GenerateIPLists(iplIps, conf)
}

type prefixIPs struct {
	prefix model.IPAMPrefix
	ips    []model.IPAddress
}

func getIPsForPrefix(nc netbox.Client, prefix model.IPAMPrefix, ch chan prefixIPs) {
	//logger.Println(fmt.Sprintf("Getting ip addresses in %s", prefix.Prefix))
	addresses := nc.GetIPAddressesByPrefix(prefix)

	ch <- prefixIPs{
		prefix: prefix,
		ips:    addresses,
	}
}

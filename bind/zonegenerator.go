package bind

import (
	"fmt"
	"strings"

	"github.com/jmesserli/netbox-to-bind/netbox"
)

type SOAInfo struct {
	// 	$TTL    86400
	// @       IN      SOA     vm-ns-1.pegnu.net.      postmaster.peg.nu. (
	//         2019021601  ; Serial
	//         3600            ; Refresh
	//         1800            ; Retry
	//         86400           ; Expire
	//         600 )           ; Negative Cache TTL

	// ; Nameserver
	// @                   IN  NS  vm-ns-1
	// vm-ns-1             IN  A   172.20.20.28

	NameserverFQDN        string
	NameserverIP          string
	DottedMailResponsible string
	TTL                   int
	Refresh               int
	Retry                 int
	Expire                int
	BindDefaultRRTTL      int
}

func applyZoneFlattening(address *netbox.IPAddress) {
	parts := strings.Split(address.GenOptions.ForwardZoneName, ".")

	if len(parts) <= 2 {
		return
	}

	zoneName := strings.Join(parts[len(parts)-2:], ".")

	oldName := address.Name
	if strings.HasSuffix(oldName, address.GenOptions.ForwardZoneName) {
		lastIdx := len(oldName) - len(address.GenOptions.ForwardZoneName) - 1
		address.Name = oldName[:lastIdx]
	}

	name := fmt.Sprintf("%s.%s", address.Name, strings.Join(parts[:len(parts)-2], "."))

	fmt.Printf("%s:\n    Original Name: %s\n    Original Zone: %s\n    New Name: %s\n    New Zone: %s\n", address.Address, oldName, address.GenOptions.ForwardZoneName, name, zoneName)

	address.GenOptions.ForwardZoneName = zoneName
	address.Name = name
}

func GenerateZones(addresses []netbox.IPAddress, soaInfo SOAInfo) {
	for _, address := range addresses {
		applyZoneFlattening(&address)
	}
}

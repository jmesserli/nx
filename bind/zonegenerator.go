package bind

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/jmesserli/netbox-to-bind/netbox"
)

// SOAInfo contains all the information to write the SOA record
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
	Serial                string
}

func applyZoneFlattening(address *netbox.IPAddress) {
	address.Name = strings.Split(address.Name, " ")[0]

	parts := strings.Split(address.GenOptions.ForwardZoneName, ".")
	if len(parts) < 2 {
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

type rrType string

const (
	// A represents the RR type "A" for a single IPv4 address
	A rrType = "A"
	// Aaaa represents the RR type "AAAA" for a single IPv6 address
	Aaaa rrType = "AAAA"
	// CName represents the RR type "CNAME" for an alias
	CName rrType = "CNAME"
	// Ptr represents the RR type "PTR" for a reverse entry
	Ptr rrType = "PTR"
)

type resourceRecord struct {
	Name  string
	Type  rrType
	RData string
}

type templateArguments struct {
	SOAInfo     SOAInfo
	Records     []resourceRecord
	GeneratedAt string
}

func putMap(theMap map[string][]resourceRecord, key string, value resourceRecord) {
	existingSlice, ok := theMap[key]

	if ok {
		theMap[key] = append(existingSlice, value)
	} else {
		theMap[key] = []resourceRecord{value}
	}
}

func GenerateZones(addresses []netbox.IPAddress, soaInfo SOAInfo) {
	t := time.Now()

	if len(soaInfo.Serial) == 0 {
		soaInfo.Serial = fmt.Sprintf("%04d%02d%02d%02d%02d%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	}

	var zoneRecordsMap = make(map[string][]resourceRecord)
	for _, address := range addresses {
		if !address.GenOptions.Enabled || !(address.GenOptions.ReverseEnabled || address.GenOptions.ForwardEnabled) {
			continue
		}

		applyZoneFlattening(&address)

		if address.GenOptions.ForwardEnabled && len(address.GenOptions.ForwardZoneName) > 0 {
			ip, _, _ := net.ParseCIDR(address.Address)

			putMap(zoneRecordsMap, address.GenOptions.ForwardZoneName, resourceRecord{
				Name:  address.Name,
				Type:  A,
				RData: ip.String(),
			})

			for _, cname := range address.GenOptions.CNames {
				putMap(zoneRecordsMap, address.GenOptions.ForwardZoneName, resourceRecord{
					Name:  cname,
					Type:  CName,
					RData: fmt.Sprintf("%s.%s.", address.Name, address.GenOptions.ForwardZoneName),
				})
			}
		}
	}

	templateArgs := templateArguments{
		SOAInfo:     soaInfo,
		GeneratedAt: t.Format(time.RFC3339),
	}

	templateString, err := ioutil.ReadFile("./templates/zone.tmpl")
	if err != nil {
		panic(err)
	}
	zoneTemplate := template.Must(template.New("zone").Parse(string(templateString)))

	for zone, records := range zoneRecordsMap {
		templateArgs.Records = records

		fmt.Println("Printing zone", zone)
		zoneTemplate.Execute(os.Stdout, templateArgs)
	}
}

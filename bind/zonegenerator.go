package bind

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/jmesserli/netbox-to-bind/netbox"
	"github.com/jmesserli/netbox-to-bind/util"
)

var logger = log.New(os.Stdout, "[generator] ", log.LstdFlags)

// SOAInfo contains all the information to write the SOA record
type SOAInfo struct {
	NameserverFQDN        string
	DottedMailResponsible string
	TTL                   int
	Refresh               int
	Retry                 int
	Expire                int
	BindDefaultRRTTL      int
	Serial                string
}

func FixFlattenAddress(address *netbox.IPAddress) {
	originalName := address.Name
	// remove everyting after the first space
	address.Name = strings.Split(strings.ToLower(originalName), " ")[0]

	originalZone := address.GenOptions.ForwardZoneName
	if len(originalZone) == 0 {
		return
	}
	zoneParts := strings.Split(originalZone, ".")
	cutoff := ""
	shortZone := originalZone
	if len(zoneParts) > 2 {
		// cut off everything that is before the last 2 zone parts
		cutoff = strings.Join(zoneParts[:len(zoneParts)-2], ".")
		shortZone = originalZone[len(cutoff)+1:]
	}
	address.GenOptions.ForwardZoneName = shortZone

	if strings.HasSuffix(address.Name, originalZone) {
		// remove suffix from name
		address.Name = address.Name[:len(address.Name)-len(originalZone)-1]
	}

	if len(cutoff) > 0 {
		// append the zone cutoff to the name
		address.Name = fmt.Sprintf("%s.%s", address.Name, cutoff)
	}

	logger.Printf("(%s).%s -> (%s).%s\n", originalName, originalZone, address.Name, shortZone)
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
	ZoneName    string
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

func ipToNibble(cidr string, minimal bool) string {
	ip, net, _ := net.ParseCIDR(cidr)
	isIP4 := (strings.Count(ip.String(), ":") < 2)

	if isIP4 {
		split := strings.Split(ip.String(), ".")
		reverse := util.ReverseSlice(split)
		if minimal {
			prefixSize, _ := net.Mask.Size()
			prefixParts := prefixSize / 8
			reverse = reverse[len(reverse)-prefixParts:]
		}
		joined := strings.Join(reverse, ".")
		return fmt.Sprintf("%s.in-addr.arpa", joined)
	}
	// else IPv6
	split := strings.Split(util.ExpandIPv6(ip), "")
	reverse := util.ReverseSlice(split)
	if minimal {
		prefixSize, _ := net.Mask.Size()
		prefixParts := prefixSize / 4
		reverse = reverse[len(reverse)-prefixParts:]
	}
	return fmt.Sprintf("%s.ip6.arpa", strings.Join(reverse, "."))
}

// GenerateZones generates the BIND zonefiles
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

		FixFlattenAddress(&address)

		ip, _, _ := net.ParseCIDR(address.Address)
		isIP4 := (strings.Count(ip.String(), ":") < 2)

		if address.GenOptions.ForwardEnabled && len(address.GenOptions.ForwardZoneName) > 0 {
			var recordType rrType
			if isIP4 {
				recordType = A
			} else {
				recordType = Aaaa
			}

			putMap(zoneRecordsMap, address.GenOptions.ForwardZoneName, resourceRecord{
				Name:  address.Name,
				Type:  recordType,
				RData: ip.String(),
			})

			for _, cname := range address.GenOptions.CNames {
				putMap(zoneRecordsMap, address.GenOptions.ForwardZoneName, resourceRecord{
					Name:  cname,
					Type:  CName,
					RData: address.Name,
				})
			}
		}

		if address.GenOptions.ReverseEnabled && len(address.GenOptions.ReverseZoneName) > 0 {
			zoneName := ipToNibble(address.GenOptions.ReverseZoneName, true)

			name := ipToNibble(address.Address, false)
			name = name[:len(name)-len(zoneName)-1]
			putMap(zoneRecordsMap, zoneName, resourceRecord{
				Name:  name,
				Type:  Ptr,
				RData: fmt.Sprintf("%s.%s.", address.Name, address.GenOptions.ForwardZoneName),
			})
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

	// clean zone directory
	dir, err := ioutil.ReadDir("./zones")
	if err != nil {
		panic(err)
	}
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{".", "zones", d.Name()}...))
	}
	for zone, records := range zoneRecordsMap {
		templateArgs.Records = records
		templateArgs.ZoneName = zone

		f, err := os.Create(fmt.Sprintf("./zones/%s.db", zone))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		w := tabwriter.NewWriter(f, 2, 2, 2, ' ', 0)
		zoneTemplate.Execute(w, templateArgs)
		w.Flush()
	}
}

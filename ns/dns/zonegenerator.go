package dns

import (
	"fmt"
	"log"
	"net"
	"os"
	"peg.nu/nx/model"
	"regexp"
	"strings"
	"text/template"
	"time"

	"peg.nu/nx/cache"
	"peg.nu/nx/config"
	"peg.nu/nx/tagparser"
	"peg.nu/nx/util"
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

type DNSIP struct {
	IP *model.IPAddress

	Enabled         bool     `nx:"enable,ns:dns"`
	ReverseZoneName string   `nx:"reverse_zone,ns:dns"`
	ForwardZoneName string   `nx:"forward_zone,ns:dns"`
	CNames          []string `nx:"cname,ns:dns"`
}

var unknownNameCounter = 1

func FixFlattenAddress(address *DNSIP) {
	originalName := address.IP.GetName()
	// remove everything after the first space
	address.IP.DnsName = strings.Split(strings.ToLower(originalName), " ")[0]
	if len(address.IP.GetName()) == 0 {
		address.IP.DnsName = fmt.Sprintf("unknown-static-%v", unknownNameCounter)
		unknownNameCounter++
	}

	originalZone := address.ForwardZoneName
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
	address.ForwardZoneName = shortZone

	if strings.HasSuffix(address.IP.GetName(), originalZone) {
		// remove suffix from name
		address.IP.DnsName = address.IP.GetName()[:len(address.IP.GetName())-len(originalZone)-1]
	}

	if len(cutoff) > 0 {
		// append the zone cutoff to the name
		address.IP.DnsName = fmt.Sprintf("%s.%s", address.IP.GetName(), cutoff)
	}

	//logger.Printf("%s -> (%s).%s\n", originalName, address.IP.Name, shortZone)
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
	Includes    []string
}

func putMap(theMap map[string][]resourceRecord, key string, value resourceRecord) {
	existingSlice, ok := theMap[key]

	if ok {
		theMap[key] = append(existingSlice, value)
	} else {
		theMap[key] = []resourceRecord{value}
	}
}

func ipToNibble(cidr string, minimal bool) (string, bool, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", false, err
	}
	isIP4 := strings.Count(ip.String(), ":") < 2

	if isIP4 {
		split := strings.Split(ip.String(), ".")
		reverse := util.ReverseSlice(split)
		if minimal {
			prefixSize, _ := ipNet.Mask.Size()
			prefixParts := prefixSize / 8
			reverse = reverse[len(reverse)-prefixParts:]
		}
		joined := strings.Join(reverse, ".")
		return fmt.Sprintf("%s.in-addr.arpa", joined), isIP4, nil
	}
	// else IPv6
	split := strings.Split(util.ExpandIPv6(ip), "")
	reverse := util.ReverseSlice(split)
	if minimal {
		prefixSize, _ := ipNet.Mask.Size()
		prefixParts := prefixSize / 4
		reverse = reverse[len(reverse)-prefixParts:]
	}
	return fmt.Sprintf("%s.ip6.arpa", strings.Join(reverse, ".")), isIP4, nil
}

// GenerateZones generates the BIND zonefiles
func GenerateZones(addresses []model.IPAddress, defaultSoaInfo SOAInfo, conf *config.NXConfig) []string {
	t := time.Now()

	if len(defaultSoaInfo.Serial) == 0 {
		atMidnight := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).Unix()
		iteration := (t.Unix() - atMidnight) / (60 * 2)

		defaultSoaInfo.Serial = fmt.Sprintf("%02d%02d%02d%03d", t.Year()-2000, t.Month(), t.Day(), iteration)
	}

	var zoneRecordsMap = make(map[string][]resourceRecord)
	for _, address := range addresses {
		dnsIP := DNSIP{IP: &address}
		tagparser.ParseTags(&dnsIP, address.Tags, address.Prefix.Tags)

		if !dnsIP.Enabled {
			continue
		}

		FixFlattenAddress(&dnsIP)

		ip, _, _ := net.ParseCIDR(address.Address)
		isIP4 := strings.Count(ip.String(), ":") < 2

		if len(dnsIP.ForwardZoneName) > 0 {
			var recordType rrType
			if isIP4 {
				recordType = A
			} else {
				recordType = Aaaa
			}

			putMap(zoneRecordsMap, dnsIP.ForwardZoneName, resourceRecord{

				Name:  address.GetName(),
				Type:  recordType,
				RData: ip.String(),
			})

			for _, cname := range dnsIP.CNames {
				putMap(zoneRecordsMap, dnsIP.ForwardZoneName, resourceRecord{
					Name:  cname,
					Type:  CName,
					RData: address.GetName(),
				})
			}
		}

		if len(dnsIP.ReverseZoneName) > 0 {
			zoneName, reverseZoneV4, err := ipToNibble(dnsIP.ReverseZoneName, true)
			if err != nil {
				logger.Printf("Could not parse cidr <%v> of %v", dnsIP.ReverseZoneName, address)
				continue
			}

			if len(dnsIP.ForwardZoneName) == 0 {
				// Parse parent tags to restore forward zone name
				tagparser.ParseTags(&dnsIP, address.Prefix.Tags, []model.Tag{})
			}

			name, addressV4, err := ipToNibble(address.Address, false)
			if err != nil {
				logger.Printf("Could not parse cidr <%v> of %v", address.Address, address)
				continue
			}

			if reverseZoneV4 != addressV4 {
				logger.Printf("IP to reverse zone family mismatch! IP: %v (isV4: %v), Reverse Zone: %v (isV4: %v)", address.Address, addressV4, dnsIP.ReverseZoneName, reverseZoneV4)
				continue
			}

			name = name[:len(name)-len(zoneName)-1]

			rData := fmt.Sprintf("%s.%s", address.GetName(), dnsIP.ForwardZoneName)
			rData = strings.TrimRight(rData, ".")
			rData = rData + "."
			putMap(zoneRecordsMap, zoneName, resourceRecord{
				Name:  name,
				Type:  Ptr,
				RData: rData,
			})
		}
	}

	templateArgs := templateArguments{
		GeneratedAt: t.Format(time.RFC3339),
	}

	templateString, err := os.ReadFile("templates/bind-zone.tmpl")
	if err != nil {
		panic(err)
	}
	zoneTemplate := template.Must(template.New("zone").Parse(string(templateString)))

	cw := cache.New("generated/hashes/zones.json")
	ignoreRegex := regexp.MustCompile("(?m)^(\\s+\\d+\\s+; serial.*|; Generated at .*)$")

	for zone, records := range zoneRecordsMap {
		templateArgs.Records = records
		templateArgs.ZoneName = zone

		soaInfo := defaultSoaInfo
		masterConf := util.FindMasterForZone(*conf, zone)
		if masterConf != nil {
			soaInfo.DottedMailResponsible = masterConf.DottedEmail
			soaInfo.NameserverFQDN = fmt.Sprintf("%s.", masterConf.Name)

			var includes []string
			for _, include := range masterConf.Includes {
				if include.Zone == zone {
					includes = append(includes, include.IncludeFiles...)
				}
			}
			templateArgs.Includes = includes
		}
		templateArgs.SOAInfo = soaInfo

		_, err := cw.WriteTemplate(
			fmt.Sprintf("generated/zones/%s.db", zone),
			zoneTemplate,
			templateArgs,
			[]*regexp.Regexp{ignoreRegex},
			true,
		)
		if err != nil {
			panic(err)
		}
	}

	util.CleanDirectoryExcept("generated/zones", cw.ProcessedFiles, conf)
	conf.UpdatedFiles = append(conf.UpdatedFiles, cw.UpdatedFiles...)

	zones := make([]string, 0, len(zoneRecordsMap))
	for key := range zoneRecordsMap {
		zones = append(zones, key)
	}
	return zones
}

package dns

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"text/template"
	"time"

	"peg.nu/nx/cache"
	"peg.nu/nx/config"
	"peg.nu/nx/util"
)

type zoneType string

const (
	zoneSecondary zoneType = "slave"
	zonePrimary   zoneType = "master"
)

type templateZone struct {
	Name            string
	Type            zoneType
	IsSecondary     bool
	IsDnssecEnabled bool
	PrimaryIP       string
	PrimaryPort     int
	TransferAcls    []string
	NotifyPrimaries []string
}

type aclPrimaryType string

const (
	listAcl       aclPrimaryType = "acl"
	listPrimaries aclPrimaryType = "masters"
)

type primaryIPAndPort struct {
	IP   string
	Port int
}

type aclPrimaryList struct {
	Type    aclPrimaryType
	Name    string
	Entries []primaryIPAndPort
}

type configTemplateVars struct {
	ServerName      string
	ServerIP        string
	GeneratedAt     string
	Zones           []templateZone
	AclPrimaryLists []aclPrimaryList
}

const defaultAclName = "nx-secondary-acl"
const defaultPrimariesName = "nx-secondary-primaries"

func generateStandardAclPrimaryLists(primaries []primaryIPAndPort) []aclPrimaryList {
	return []aclPrimaryList{
		{Type: listAcl, Name: defaultAclName, Entries: primaries},
		{Type: listPrimaries, Name: defaultPrimariesName, Entries: primaries},
	}
}

func canonicalizeZoneName(zone string) string {
	hasher := md5.New()
	hasher.Write([]byte(zone))
	return hex.EncodeToString(hasher.Sum(nil))[:8]
}

func getAdditionalAclPrimaryName(zone string, ty aclPrimaryType) string {
	canonicalZone := canonicalizeZoneName(zone)

	return fmt.Sprintf("nx-secondary-%s-%s", ty, canonicalZone)
}

func generateAdditionalAclPrimaryLists(primaryConfig *config.PrimaryConfig) []aclPrimaryList {
	var lists []aclPrimaryList

	if primaryConfig.AdditionalSecondaries != nil {
		for zone, secondaries := range primaryConfig.AdditionalSecondaries {
			var secondariesWithPorts []primaryIPAndPort
			for _, secondary := range secondaries {
				secondariesWithPorts = append(secondariesWithPorts, primaryIPAndPort{IP: secondary}) // port is empty for now
			}

			lists = append(lists, []aclPrimaryList{
				{Type: listAcl, Name: getAdditionalAclPrimaryName(zone, listAcl), Entries: secondariesWithPorts},
				{Type: listPrimaries, Name: getAdditionalAclPrimaryName(zone, listPrimaries), Entries: secondariesWithPorts},
			}...)
		}
	}

	return lists
}

func GenerateConfigs(zones []string, conf *config.NXConfig) {
	templateString, err := os.ReadFile("templates/bind-config.tmpl")
	if err != nil {
		panic(err)
	}
	configTemplate := template.Must(template.New("config").Parse(string(templateString)))
	ignoreRegexes := []*regexp.Regexp{
		regexp.MustCompile("(?m)^ \\* Generated at.*$"),
	}
	cw := cache.New(configTemplate, ignoreRegexes, false)

	templateVars := configTemplateVars{
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	for _, currentPrimary := range conf.Namespaces.DNS.Primaries {
		templateVars.ServerName = currentPrimary.Name
		templateVars.ServerIP = currentPrimary.IP
		var templateZones []templateZone

		for _, zonesPrimary := range conf.Namespaces.DNS.Primaries {
			isPrimary := zonesPrimary.Name == currentPrimary.Name
			serverZoneType := zonePrimary
			if !isPrimary {
				serverZoneType = zoneSecondary
			}

			for _, zone := range zonesPrimary.Zones {
				if !util.SliceContainsString(zones, zone) {
					continue
				}

				transferAcls := []string{defaultAclName}
				notifyPrimaries := []string{defaultPrimariesName}

				_, hasAdditionalSecondaries := currentPrimary.AdditionalSecondaries[zone]
				if hasAdditionalSecondaries {
					transferAcls = append(transferAcls, getAdditionalAclPrimaryName(zone, listAcl))
					notifyPrimaries = append(notifyPrimaries, getAdditionalAclPrimaryName(zone, listPrimaries))
				}

				dnssecEnabled := util.SliceContainsString(zonesPrimary.DnssecZones, zone)

				templateZones = append(templateZones, templateZone{
					IsSecondary:     !isPrimary,
					IsDnssecEnabled: dnssecEnabled,
					PrimaryIP:       zonesPrimary.IP,
					PrimaryPort:     zonesPrimary.Port,
					Name:            zone,
					Type:            serverZoneType,
					TransferAcls:    transferAcls,
					NotifyPrimaries: notifyPrimaries,
				})
			}
		}

		templateVars.Zones = templateZones

		var primaryIpsWithoutCurrent = make([]primaryIPAndPort, 0, len(conf.Namespaces.DNS.Primaries)-1)
		for _, primary := range conf.Namespaces.DNS.Primaries {
			if primary.IP != currentPrimary.IP {
				primaryIpsWithoutCurrent = append(primaryIpsWithoutCurrent, primaryIPAndPort{IP: primary.IP, Port: primary.Port})
			}
		}

		aclPrimaryLists := generateStandardAclPrimaryLists(primaryIpsWithoutCurrent)
		aclPrimaryLists = append(aclPrimaryLists, generateAdditionalAclPrimaryLists(&currentPrimary)...)
		templateVars.AclPrimaryLists = aclPrimaryLists

		_, err = cw.WriteTemplate(
			fmt.Sprintf("generated/bind-config/%s.conf", currentPrimary.Name),
			templateVars,
		)
		if err != nil {
			panic(err)
		}
	}

	util.CleanDirectoryExcept("generated/bind-config", cw.ProcessedFiles, conf)
	conf.UpdatedFiles = append(conf.UpdatedFiles, cw.UpdatedFiles...)
}

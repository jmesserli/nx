package dns

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"regexp"
	"text/template"
	"time"

	"peg.nu/nx/cache"
	"peg.nu/nx/config"
	"peg.nu/nx/util"
)

type zoneType string

const (
	zoneSlave  zoneType = "slave"
	zoneMaster zoneType = "master"
)

type templateZone struct {
	Name          string
	Type          zoneType
	IsSlave       bool
	MasterIP      string
	TransferAcls  []string
	NotifyMasters []string
}

type aclMasterType string

const (
	listAcl    aclMasterType = "acl"
	listMaster aclMasterType = "masters"
)

type aclMastersList struct {
	Type    aclMasterType
	Name    string
	Entries []string
}

type configTemplateVars struct {
	ServerName     string
	ServerIP       string
	GeneratedAt    string
	Zones          []templateZone
	AclMasterLists []aclMastersList
}

const defaultAclName = "nx-slaves-acl"
const defaultMastersName = "nx-slaves-masters"

func generateStandardAclMasterLists(masterIps []string) []aclMastersList {
	return []aclMastersList{
		{Type: listAcl, Name: defaultAclName, Entries: masterIps},
		{Type: listMaster, Name: defaultMastersName, Entries: masterIps},
	}
}

func canonicalizeZoneName(zone string) string {
	hasher := md5.New()
	hasher.Write([]byte(zone))
	return hex.EncodeToString(hasher.Sum(nil))[:8]
}

func getAdditionalAclMasterName(zone string, ty aclMasterType) string {
	canonicalZone := canonicalizeZoneName(zone)

	return fmt.Sprintf("nx-slaves-%s-%s", ty, canonicalZone)
}

func generateAdditionalAclMasterLists(masterConfig *config.MasterConfig) []aclMastersList {
	var lists []aclMastersList

	if masterConfig.AdditionalSlaves != nil {
		for zone, slaves := range masterConfig.AdditionalSlaves {
			lists = append(lists, []aclMastersList{
				{Type: listAcl, Name: getAdditionalAclMasterName(zone, listAcl), Entries: slaves},
				{Type: listMaster, Name: getAdditionalAclMasterName(zone, listMaster), Entries: slaves},
			}...)
		}
	}

	return lists
}

func GenerateConfigs(zones []string, conf *config.NXConfig) {
	templateString, err := ioutil.ReadFile("templates/bind-config.tmpl")
	if err != nil {
		panic(err)
	}
	configTemplate := template.Must(template.New("config").Parse(string(templateString)))
	cw := cache.New("generated/hashes/bind-config.json")
	ignoreRegex := regexp.MustCompile("(?m)^ \\* Generated at.*$")

	templateVars := configTemplateVars{
		GeneratedAt: time.Now().Format(time.RFC3339),
	}
	for _, currentMaster := range conf.Namespaces.DNS.Masters {
		templateVars.ServerName = currentMaster.Name
		templateVars.ServerIP = currentMaster.IP
		var templateZones []templateZone

		for _, zonesMaster := range conf.Namespaces.DNS.Masters {
			isMaster := zonesMaster.Name == currentMaster.Name
			masterZoneType := zoneMaster
			if !isMaster {
				masterZoneType = zoneSlave
			}

			for _, zone := range zonesMaster.Zones {
				if !util.SliceContainsString(zones, zone) {
					continue
				}

				transferAcls := []string{defaultAclName}
				notifyMasters := []string{defaultMastersName}

				_, hasAdditionalSlaves := currentMaster.AdditionalSlaves[zone]
				if hasAdditionalSlaves {
					transferAcls = append(transferAcls, getAdditionalAclMasterName(zone, listAcl))
					notifyMasters = append(notifyMasters, getAdditionalAclMasterName(zone, listMaster))
				}

				templateZones = append(templateZones, templateZone{
					IsSlave:       !isMaster,
					MasterIP:      zonesMaster.IP,
					Name:          zone,
					Type:          masterZoneType,
					TransferAcls:  transferAcls,
					NotifyMasters: notifyMasters,
				})
			}
		}

		templateVars.Zones = templateZones

		var masterIPsWithoutCurrent = make([]string, 0, len(conf.Namespaces.DNS.Masters)-1)
		for _, master := range conf.Namespaces.DNS.Masters {
			if master.IP != currentMaster.IP {
				masterIPsWithoutCurrent = append(masterIPsWithoutCurrent, master.IP)
			}
		}

		aclMastersLists := generateStandardAclMasterLists(masterIPsWithoutCurrent)
		aclMastersLists = append(aclMastersLists, generateAdditionalAclMasterLists(&currentMaster)...)
		templateVars.AclMasterLists = aclMastersLists

		_, err = cw.WriteTemplate(
			fmt.Sprintf("generated/bind-config/%s.conf", currentMaster.Name),
			configTemplate,
			templateVars,
			[]*regexp.Regexp{ignoreRegex},
			false,
		)
		if err != nil {
			panic(err)
		}
	}

	util.CleanDirectoryExcept("generated/bind-config", cw.ProcessedFiles, conf)
	conf.UpdatedFiles = append(conf.UpdatedFiles, cw.UpdatedFiles...)
}

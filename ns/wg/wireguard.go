package wg

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/template"

	"github.com/jmesserli/nx/netbox"
	"github.com/jmesserli/nx/tagparser"
	"github.com/jmesserli/nx/util"
)

type templatePeer struct {
	Name      string
	PublicKey string `nx:"pubkey,ns:wg"`
	IP        string `nx:"ip,ns:wg"`
	Port      string `nx:"port,ns:wg"`
}

type templateData struct {
	ServerName string
	OwnAddress string
	OwnPort    string
	Peers      []templatePeer
}

type parsedIp struct {
	IP   netbox.IPAddress
	peer templatePeer
}

func putMap(theMap map[string][]parsedIp, key string, value parsedIp) {
	existingSlice, ok := theMap[key]

	if ok {
		theMap[key] = append(existingSlice, value)
	} else {
		theMap[key] = []parsedIp{value}
	}
}

func GenerateWgConfigs(ips []netbox.IPAddress) {
	var vpnPeers = make(map[string][]parsedIp, 0)

	// find and parse valid peers
	for _, ip := range ips {
		peer := templatePeer{}
		tagparser.ParseTags(&peer, ip.Tags, ip.Prefix.Tags)

		if len(peer.PublicKey) == 0 || len(peer.IP) == 0 || len(peer.Port) == 0 {
			continue
		}

		peer.Name = ip.Name
		putMap(vpnPeers, ip.Prefix.EnOptions.WGVpnName, parsedIp{IP: ip, peer: peer})
	}

	templateString, err := ioutil.ReadFile("./templates/wg-config.tmpl")
	if err != nil {
		panic(err)
	}
	zoneTemplate := template.Must(template.New("wg-config").Parse(string(templateString)))
	util.CleanDirectory("./generated/wg")

	for vpnName, peers := range vpnPeers {
		for _, peer := range peers {
			var peersWithoutCurrent = make([]templatePeer, 0, len(vpnPeers)-1)
			for _, currentPeer := range peers {
				if peer.peer.PublicKey != currentPeer.peer.PublicKey {
					peersWithoutCurrent = append(peersWithoutCurrent, currentPeer.peer)
				}
			}

			data := templateData{
				OwnAddress: peer.IP.Address,
				OwnPort:    peer.peer.Port,
				ServerName: peer.IP.Name,
				Peers:      peersWithoutCurrent,
			}

			f, err := os.Create(fmt.Sprintf("./generated/wg/%s_%s.conf", vpnName, data.ServerName))
			if err != nil {
				panic(err)
			}
			defer f.Close()

			zoneTemplate.Execute(f, data)
		}
	}

}

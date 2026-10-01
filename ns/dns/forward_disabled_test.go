package dns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"peg.nu/nx/config"
	"peg.nu/nx/model"
	"peg.nu/nx/resolver"
)

func TestForwardDisabledPreservesPTR(t *testing.T) {
	templateContent, err := os.ReadFile(filepath.Join("..", "..", "templates", "bind-zone.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name                            string
		prefixDisabled, addressDisabled *bool
		wantForward                     bool
		legacyClearing                  bool
	}{
		{"forward by default", nil, nil, true, false},
		{"disable prefix", new(true), nil, false, false},
		{"disable individual IPs", nil, new(true), false, false},
		{"restore individual IPs", new(true), new(false), true, false},
		{"legacy clearing normalizes to disable flag", nil, nil, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.MkdirAll("templates", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll("generated/zones", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("templates/bind-zone.tmpl", templateContent, 0o644); err != nil {
				t.Fatal(err)
			}
			prefixV4 := model.IPAMPrefix{Prefix: "192.0.2.0/24", CustomFields: model.CustomFields{DNSForwardDisabled: tt.prefixDisabled}, Tags: []model.Tag{{Name: "nx:dns:enable[true]"}, {Name: "nx:dns:forward_zone[dev.example.com]"}, {Name: "nx:dns:reverse_zone[192.0.2.0/24]"}}}
			prefixV6 := model.IPAMPrefix{Prefix: "2001:db8::/64", CustomFields: model.CustomFields{DNSForwardDisabled: tt.prefixDisabled}, Tags: []model.Tag{{Name: "nx:dns:enable[true]"}, {Name: "nx:dns:forward_zone[dev.example.com]"}, {Name: "nx:dns:reverse_zone[2001:db8::/64]"}}}
			resolver.ResolvePrefix(&prefixV4)
			resolver.ResolvePrefix(&prefixV6)
			addresses := []model.IPAddress{
				{Address: "192.0.2.10/24", DnsName: "Host4.Dev.Example.com", Prefix: &prefixV4, CustomFields: model.CustomFields{DNSForwardDisabled: tt.addressDisabled}, Tags: []model.Tag{{Name: "nx:dns:cname[alias4]"}}},
				{Address: "2001:db8::10/64", DnsName: "Host6.Dev.Example.com", Prefix: &prefixV6, CustomFields: model.CustomFields{DNSForwardDisabled: tt.addressDisabled}, Tags: []model.Tag{{Name: "nx:dns:cname[alias6]"}}},
			}
			for i := range addresses {
				if tt.legacyClearing {
					addresses[i].Tags = append(addresses[i].Tags, model.Tag{Name: "nx:dns:forward_zone[]"})
				}
				resolver.ResolveAddress(&addresses[i])
			}
			conf := config.NXConfig{}
			soa := SOAInfo{Serial: "261001001", NameserverFQDN: "ns.example.com.", DottedMailResponsible: "hostmaster.example.com"}
			zones := GenerateZones(addresses, soa, &conf)
			if slices.Contains(zones, "example.com") != tt.wantForward {
				t.Fatalf("unexpected forward zone presence: %v", zones)
			}
			assertZoneLines(t, "generated/zones/2.0.192.in-addr.arpa.db", "10 PTR host4.dev.example.com.")
			v6zone, _, err := ipToNibble("2001:db8::/64", true)
			if err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join("generated", "zones", v6zone+".db"))
			if err != nil || !strings.Contains(string(body), "host6.dev.example.com.") {
				t.Fatalf("missing IPv6 PTR: %s, %v", body, err)
			}
			body, err = os.ReadFile("generated/search-index.json")
			if err != nil {
				t.Fatal(err)
			}
			var index searchIndex
			if err := json.Unmarshal(body, &index); err != nil {
				t.Fatal(err)
			}
			if tt.wantForward {
				assertZoneLines(t, "generated/zones/example.com.db", "host4.dev A 192.0.2.10", "host6.dev AAAA 2001:db8::10", "alias4 CNAME host4.dev", "alias6 CNAME host6.dev")
				if len(index.Records) != 2 || len(index.Zones) != 1 {
					t.Fatalf("unexpected index: %+v", index)
				}
			} else if len(index.Records) != 0 || len(index.Zones) != 0 {
				t.Fatalf("disabled forward records in index: %+v", index)
			}
		})
	}
}

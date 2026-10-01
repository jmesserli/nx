package resolver

import (
	"encoding/json"
	"reflect"
	"testing"

	"peg.nu/nx/model"
)

func TestResolution(t *testing.T) {
	var prefix model.IPAMPrefix
	if err := json.Unmarshal([]byte(`{"tags":[{"name":"nx:dns:enable[true]"},{"name":"nx:dns:forward_zone[tag.example]"},{"name":"nx:dns:reverse_zone[192.0.2.0/24]"},{"name":"nx:dns:cname[parent]"},{"name":"nx:ipl:enable[true]"},{"name":"nx:ipl:list[parent]"}],"custom_fields":{"nx_dns_forward_zone":"cf.example","nx_ip_lists":["custom"]}}`), &prefix); err != nil {
		t.Fatal(err)
	}
	ResolvePrefix(&prefix)
	tests := []struct {
		name, body string
		want       model.Configuration
	}{
		{"inherit", `{}`, model.Configuration{DNSEnabled: true, DNSForwardZone: "cf.example", DNSReverseZone: "192.0.2.0/24", DNSCNames: []string{"parent"}, IPLEnabled: true, IPLists: []string{"custom"}}},
		{"null inherits", `{"custom_fields":{"nx_dns_enabled":null,"nx_dns_forward_zone":null,"nx_dns_cnames":null,"nx_ip_lists":null}}`, model.Configuration{DNSEnabled: true, DNSForwardZone: "cf.example", DNSReverseZone: "192.0.2.0/24", DNSCNames: []string{"parent"}, IPLEnabled: true, IPLists: []string{"custom"}}},
		{"address tags beat prefix fields", `{"tags":[{"name":"nx:dns:enable[false]"},{"name":"nx:dns:forward_zone[address.example]"},{"name":"nx:ipl:list[local]"}]}`, model.Configuration{DNSForwardZone: "address.example", DNSReverseZone: "192.0.2.0/24", DNSCNames: []string{"parent"}, IPLEnabled: true, IPLists: []string{"local"}}},
		{"explicit fields beat tags and clear", `{"tags":[{"name":"nx:dns:enable[true]"},{"name":"nx:dns:forward_zone[address.example]"},{"name":"nx:dns:cname[local]"},{"name":"nx:ipl:enable[true]"},{"name":"nx:ipl:list[local]"}],"custom_fields":{"nx_dns_enabled":false,"nx_dns_forward_zone":"","nx_dns_reverse_zone":"","nx_dns_cnames":[],"nx_ip_lists_enabled":false,"nx_ip_lists":[]}}`, model.Configuration{DNSForwardDisabled: true, DNSForwardZone: "cf.example", DNSCNames: []string{}, IPLists: []string{}}},
		{"legacy empty forward retains PTR zone", `{"tags":[{"name":"nx:dns:forward_zone[]"}]}`, model.Configuration{DNSEnabled: true, DNSForwardDisabled: true, DNSForwardZone: "cf.example", DNSReverseZone: "192.0.2.0/24", DNSCNames: []string{"parent"}, IPLEnabled: true, IPLists: []string{"custom"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var address model.IPAddress
			if err := json.Unmarshal([]byte(tt.body), &address); err != nil {
				t.Fatal(err)
			}
			address.Prefix = &prefix
			ResolveAddress(&address)
			if !reflect.DeepEqual(address.Config, tt.want) {
				t.Fatalf("got %+v, want %+v", address.Config, tt.want)
			}
			ResolveAddress(&address)
			if !reflect.DeepEqual(address.Config, tt.want) {
				t.Fatal("resolution is not repeatable")
			}
		})
	}
	var address model.IPAddress
	address.Prefix = &prefix
	ResolveAddress(&address)
	address.Config.DNSCNames[0] = "changed"
	address.Config.IPLists[0] = "changed"
	if prefix.Config.DNSCNames[0] != "parent" || prefix.Config.IPLists[0] != "custom" {
		t.Fatal("address changed prefix lists")
	}
}

func TestCustomFieldTypes(t *testing.T) {
	for _, body := range []string{
		`{"nx_dns_enabled":"false"}`, `{"nx_dns_forward_zone":42}`,
		`{"nx_dns_forward_disabled":"true"}`,
		`{"nx_dns_reverse_zone":[]}`, `{"nx_dns_cnames":"alias"}`,
		`{"nx_ip_lists_enabled":0}`, `{"nx_ip_lists":[1]}`,
	} {
		var fields model.CustomFields
		if err := json.Unmarshal([]byte(body), &fields); err == nil {
			t.Errorf("accepted malformed fields: %s", body)
		}
	}
	var address model.IPAddress
	ResolveAddress(&address)
	if !reflect.DeepEqual(address.Config, model.Configuration{}) {
		t.Fatal("missing settings must default to zero values")
	}
}

func TestForwardDisabledInheritance(t *testing.T) {
	for _, tt := range []struct {
		name, prefixFields, addressFields string
		want                              bool
	}{
		{"default generates forward", `{}`, `{}`, false},
		{"prefix disables forward", `{"nx_dns_forward_disabled":true}`, `{}`, true},
		{"null inherits disabled prefix", `{"nx_dns_forward_disabled":true}`, `{"nx_dns_forward_disabled":null}`, true},
		{"address disables forward", `{}`, `{"nx_dns_forward_disabled":true}`, true},
		{"address restores forward", `{"nx_dns_forward_disabled":true}`, `{"nx_dns_forward_disabled":false}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefix := model.IPAMPrefix{Tags: []model.Tag{{Name: "nx:dns:enable[true]"}, {Name: "nx:dns:forward_zone[example.com]"}}}
			if err := json.Unmarshal([]byte(tt.prefixFields), &prefix.CustomFields); err != nil {
				t.Fatal(err)
			}
			ResolvePrefix(&prefix)
			address := model.IPAddress{Prefix: &prefix}
			if err := json.Unmarshal([]byte(tt.addressFields), &address.CustomFields); err != nil {
				t.Fatal(err)
			}
			ResolveAddress(&address)
			if address.Config.DNSForwardDisabled != tt.want || !address.Config.DNSEnabled || address.Config.DNSForwardZone != "example.com" {
				t.Fatalf("unexpected resolved configuration: %+v", address.Config)
			}
		})
	}
}

func TestLegacyForwardClearing(t *testing.T) {
	for _, tt := range []struct {
		name, prefixJSON, addressJSON, wantZone string
		wantDisabled                            bool
	}{
		{"address clearing keeps parent zone", `{}`, `{"tags":[{"name":"nx:dns:forward_zone[]"}]}`, "example.com", true},
		{"null custom zone keeps legacy clearing", `{}`, `{"tags":[{"name":"nx:dns:forward_zone[]"}],"custom_fields":{"nx_dns_forward_zone":null}}`, "example.com", true},
		{"custom zone replaces legacy clearing", `{}`, `{"tags":[{"name":"nx:dns:forward_zone[]"}],"custom_fields":{"nx_dns_forward_zone":"custom.example"}}`, "custom.example", false},
		{"explicit false overrides legacy clearing", `{}`, `{"tags":[{"name":"nx:dns:forward_zone[]"}],"custom_fields":{"nx_dns_forward_disabled":false}}`, "example.com", false},
		{"explicit true with custom zone", `{}`, `{"tags":[{"name":"nx:dns:forward_zone[]"}],"custom_fields":{"nx_dns_forward_zone":"custom.example","nx_dns_forward_disabled":true}}`, "custom.example", true},
		{"prefix clearing inherited", `{"tags":[{"name":"nx:dns:enable[true]"},{"name":"nx:dns:forward_zone[]"}]}`, `{}`, "", true},
		{"address zone undoes prefix legacy clearing", `{"tags":[{"name":"nx:dns:enable[true]"},{"name":"nx:dns:forward_zone[]"}]}`, `{"tags":[{"name":"nx:dns:forward_zone[address.example]"}]}`, "address.example", false},
		{"address zone retains explicit prefix disable", `{"custom_fields":{"nx_dns_forward_disabled":true}}`, `{"tags":[{"name":"nx:dns:forward_zone[address.example]"}]}`, "address.example", true},
		{"address clearing overrides prefix false", `{"custom_fields":{"nx_dns_forward_disabled":false}}`, `{"tags":[{"name":"nx:dns:forward_zone[]"}]}`, "example.com", true},
		{"API empty zone normalized too", `{}`, `{"custom_fields":{"nx_dns_forward_zone":""}}`, "example.com", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefix := model.IPAMPrefix{Tags: []model.Tag{{Name: "nx:dns:enable[true]"}, {Name: "nx:dns:forward_zone[example.com]"}}}
			if err := json.Unmarshal([]byte(tt.prefixJSON), &prefix); err != nil {
				t.Fatal(err)
			}
			ResolvePrefix(&prefix)
			var address model.IPAddress
			if err := json.Unmarshal([]byte(tt.addressJSON), &address); err != nil {
				t.Fatal(err)
			}
			address.Prefix = &prefix
			ResolveAddress(&address)
			if address.Config.DNSForwardZone != tt.wantZone || address.Config.DNSForwardDisabled != tt.wantDisabled || !address.Config.DNSEnabled {
				t.Fatalf("unexpected configuration: %+v", address.Config)
			}
		})
	}
}

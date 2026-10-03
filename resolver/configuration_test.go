package resolver

import (
	"encoding/json"
	"reflect"
	"testing"

	"peg.nu/nx/model"
)

func TestResolution(t *testing.T) {
	var prefix model.IPAMPrefix
	if err := json.Unmarshal([]byte(`{"custom_fields":{"nx_dns_enabled":true,"nx_dns_forward_zone":"example.com","nx_dns_reverse_zone":"192.0.2.0/24","nx_ip_lists_enabled":true,"nx_ip_lists":["internal"]}}`), &prefix); err != nil {
		t.Fatal(err)
	}
	ResolvePrefix(&prefix)
	inherited := model.Configuration{DNSEnabled: true, DNSForwardZone: "example.com", DNSReverseZone: "192.0.2.0/24", IPLEnabled: true, IPLists: []string{"internal"}}
	for _, tt := range []struct {
		name, body string
		want       model.Configuration
	}{
		{"inherit", `{}`, inherited},
		{"null inherits", `{"custom_fields":{"nx_dns_enabled":null,"nx_dns_forward_zone":null,"nx_dns_reverse_zone":null,"nx_dns_cnames":null,"nx_ip_lists_enabled":null,"nx_ip_lists":null}}`, inherited},
		{"explicit false and empty values", `{"custom_fields":{"nx_dns_enabled":false,"nx_dns_forward_zone":"","nx_dns_reverse_zone":"","nx_dns_cnames":[],"nx_ip_lists_enabled":false,"nx_ip_lists":[]}}`, model.Configuration{DNSCNames: []string{}, IPLists: []string{}}},
		{"address overrides", `{"custom_fields":{"nx_dns_forward_zone":"address.example","nx_dns_reverse_zone":"192.0.2.0/28","nx_dns_cnames":["alias"],"nx_ip_lists":["local"]}}`, model.Configuration{DNSEnabled: true, DNSForwardZone: "address.example", DNSReverseZone: "192.0.2.0/28", DNSCNames: []string{"alias"}, IPLEnabled: true, IPLists: []string{"local"}}},
		{"unknown fields ignored", `{"custom_fields":{"unrelated":42}}`, inherited},
		{"tags ignored", `{"tags":[{"name":"nx:dns:enable[false]"},{"name":"nx:dns:forward_zone[]"},{"name":"nx:ipl:list[other]"}]}`, inherited},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var address model.IPAddress
			if err := json.Unmarshal([]byte(tt.body), &address); err != nil {
				t.Fatal(err)
			}
			address.Prefix = &prefix
			for range 2 {
				ResolveAddress(&address)
				if !reflect.DeepEqual(address.Config, tt.want) {
					t.Fatalf("got %+v, want %+v", address.Config, tt.want)
				}
			}
		})
	}
	var address model.IPAddress
	address.Prefix = &prefix
	ResolveAddress(&address)
	address.Config.IPLists[0] = "changed"
	if prefix.Config.IPLists[0] != "internal" || (*prefix.CustomFields.IPLists)[0] != "internal" {
		t.Fatal("address changed prefix lists")
	}
	address.CustomFields.DNSCNames = new([]string{"alias"})
	ResolveAddress(&address)
	address.Config.DNSCNames[0] = "changed"
	if (*address.CustomFields.DNSCNames)[0] != "alias" {
		t.Fatal("resolved configuration changed source aliases")
	}
}

func TestTagsDoNotEnablePrefix(t *testing.T) {
	var prefix model.IPAMPrefix
	if err := json.Unmarshal([]byte(`{"tags":[{"name":"nx:dns:enable[true]"},{"name":"nx:ipl:enable[true]"}]}`), &prefix); err != nil {
		t.Fatal(err)
	}
	ResolvePrefix(&prefix)
	if !reflect.DeepEqual(prefix.Config, model.Configuration{}) {
		t.Fatalf("tags affected configuration: %+v", prefix.Config)
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
		{"address zone retains prefix disable", `{"nx_dns_forward_disabled":true}`, `{"nx_dns_forward_zone":"address.example"}`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefix := model.IPAMPrefix{}
			if err := json.Unmarshal([]byte(tt.prefixFields), &prefix.CustomFields); err != nil {
				t.Fatal(err)
			}
			prefix.CustomFields.DNSEnabled = new(true)
			prefix.CustomFields.DNSForwardZone = new("example.com")
			ResolvePrefix(&prefix)
			address := model.IPAddress{Prefix: &prefix}
			if err := json.Unmarshal([]byte(tt.addressFields), &address.CustomFields); err != nil {
				t.Fatal(err)
			}
			ResolveAddress(&address)
			if address.Config.DNSForwardDisabled != tt.want || !address.Config.DNSEnabled {
				t.Fatalf("unexpected resolved configuration: %+v", address.Config)
			}
		})
	}
}

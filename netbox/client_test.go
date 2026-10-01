package netbox

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"peg.nu/nx/config"
)

func TestRESTResolvesConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Token test-token" {
			t.Error("missing API authorization")
		}
		if r.URL.Query().Get("limit") != "2000" {
			t.Error("missing request limit")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/ipam/prefixes/":
			w.Write([]byte(`{"count":1,"results":[{"id":1,"prefix":"192.0.2.0/24","tags":[{"name":"nx:dns:enable[true]"}],"custom_fields":{"nx_dns_forward_zone":"example.com","nx_ip_lists_enabled":true,"nx_ip_lists":["internal"]}}]}`))
		case "/api/ipam/ip-addresses/":
			if r.URL.Query().Get("parent") != "192.0.2.0/24" {
				t.Error("incorrect parent filter")
			}
			w.Write([]byte(`{"count":1,"results":[{"id":2,"address":"192.0.2.10/24","dns_name":"host","tags":[{"name":"nx:dns:cname[legacy]"}],"custom_fields":{"nx_dns_cnames":["alias"],"nx_ip_lists_enabled":false}}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := New(config.NXConfig{Netbox: config.NetboxConfig{URL: server.URL + "/api", ApiKey: "test-token"}})
	prefixes := client.GetIPAMPrefixes()
	if len(prefixes) != 1 || !prefixes[0].Config.DNSEnabled || !prefixes[0].Config.IPLEnabled {
		t.Fatalf("unexpected prefixes: %+v", prefixes)
	}
	addresses := client.GetIPAddressesByPrefix(prefixes[0])
	if len(addresses) != 1 {
		t.Fatalf("unexpected addresses: %+v", addresses)
	}
	address := addresses[0]
	if address.Prefix == nil || address.Config.DNSForwardZone != "example.com" || address.Config.IPLEnabled || len(address.Config.DNSCNames) != 1 || address.Config.DNSCNames[0] != "alias" {
		t.Fatalf("unexpected address: %+v", address)
	}
}

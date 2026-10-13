package dns

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"peg.nu/nx/config"
	"peg.nu/nx/model"
)

func TestRepeatedGenerationIsStable(t *testing.T) {
	templates := make(map[string][]byte)
	for _, name := range []string{"bind-zone.tmpl", "bind-config.tmpl"} {
		content, err := os.ReadFile(filepath.Join("..", "..", "templates", name))
		if err != nil {
			t.Fatal(err)
		}
		templates[name] = content
	}
	t.Chdir(t.TempDir())
	for _, dir := range []string{"templates", "generated/zones", "generated/bind-config"} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range templates {
		if err := os.WriteFile(filepath.Join("templates", name), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	conf := config.NXConfig{Namespaces: config.NamespaceConfig{DNS: config.DNSNamespaceConfig{
		Primaries: []config.PrimaryConfig{{
			Name: "ns.example.com", DottedEmail: "hostmaster.example.com",
			Zones:    []string{"2.0.192.in-addr.arpa", "3.0.192.in-addr.arpa", "4.0.192.in-addr.arpa"},
			Includes: []config.ZoneInclude{{Zone: "2.0.192.in-addr.arpa", IncludeFiles: []string{"extra.db"}}},
			AdditionalSecondaries: config.AdditionalSecondariesConfig{
				"4.0.192.in-addr.arpa": {"192.0.2.4"},
				"2.0.192.in-addr.arpa": {"192.0.2.2"},
				"3.0.192.in-addr.arpa": {"192.0.2.3"},
			},
		}},
	}}}
	// The unassigned zone includes an unnamed address to exercise counter reset.
	addresses := []model.IPAddress{
		{Address: "10.20.12.1/24", Config: model.Configuration{DNSEnabled: true, DNSReverseZone: "10.20.12.0/24"}},
		{Address: "192.0.2.1/24", DnsName: "one.example.com", Config: model.Configuration{DNSEnabled: true, DNSReverseZone: "192.0.2.0/24"}},
		{Address: "192.0.3.1/24", DnsName: "two.example.com", Config: model.Configuration{DNSEnabled: true, DNSReverseZone: "192.0.3.0/24"}},
		{Address: "192.0.4.1/24", DnsName: "three.example.com", Config: model.Configuration{DNSEnabled: true, DNSReverseZone: "192.0.4.0/24"}},
	}
	soa := SOAInfo{Serial: "261013001", NameserverFQDN: "unknown-nameserver.local.", DottedMailResponsible: `unknown\.admin.local`}
	for run := range 100 {
		conf.UpdatedFiles = nil
		zones := GenerateZones(addresses, soa, &conf)
		GenerateConfigs(zones, &conf)
		content, err := os.ReadFile("generated/zones/12.20.10.in-addr.arpa.db")
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), "$INCLUDE") || !strings.Contains(string(content), "unknown-static-1.") {
			t.Fatalf("run %d: unexpected unassigned zone:\n%s", run, content)
		}
		assigned, err := os.ReadFile("generated/zones/2.0.192.in-addr.arpa.db")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(assigned), "$INCLUDE extra.db") {
			t.Fatalf("run %d: assigned zone lost its include", run)
		}
		if run > 0 && len(conf.UpdatedFiles) != 0 {
			t.Fatalf("run %d: unchanged inputs updated %v", run, conf.UpdatedFiles)
		}
	}
}

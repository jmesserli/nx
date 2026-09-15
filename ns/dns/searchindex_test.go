package dns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"peg.nu/nx/config"
	"peg.nu/nx/model"
)

func TestGenerateZonesWritesSearchIndex(t *testing.T) {
	templateContent, err := os.ReadFile(filepath.Join("..", "..", "templates", "bind-zone.tmpl"))
	if err != nil {
		t.Fatalf("read BIND zone template: %v", err)
	}

	workingDirectory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workingDirectory, "templates"), 0o755); err != nil {
		t.Fatalf("create templates directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workingDirectory, "templates", "bind-zone.tmpl"), templateContent, 0o644); err != nil {
		t.Fatalf("copy BIND zone template: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(workingDirectory, "generated", "zones"), 0o755); err != nil {
		t.Fatalf("create generated zones directory: %v", err)
	}
	t.Chdir(workingDirectory)

	prefixV4 := &model.IPAMPrefix{
		Prefix: "192.0.2.0/24",
		Tags: []model.Tag{
			{Name: "nx:dns:enable[true]"},
			{Name: "nx:dns:forward_zone[dev.example.com]"},
		},
	}
	prefixV6 := &model.IPAMPrefix{
		Prefix: "2001:db8::/64",
		Tags: []model.Tag{
			{Name: "nx:dns:enable[true]"},
			{Name: "nx:dns:forward_zone[alpha.test]"},
		},
	}
	reversePrefix := &model.IPAMPrefix{
		Prefix: "198.51.100.0/24",
		Tags: []model.Tag{
			{Name: "nx:dns:enable[true]"},
			{Name: "nx:dns:reverse_zone[198.51.100.0/24]"},
		},
	}
	addresses := []model.IPAddress{
		{
			Address: "192.0.2.42/24",
			DnsName: "ZzZ.Dev.Example.com descriptive text",
			Tags: []model.Tag{
				{Name: "nx:dns:cname[z-alias]"},
				{Name: "nx:dns:cname[a-alias]"},
			},
			Prefix: prefixV4,
		},
		{
			Address: "2001:db8::7/64",
			DnsName: "alpha",
			Prefix:  prefixV6,
		},
		{
			Address: "198.51.100.9/24",
			DnsName: "ptr-only",
			Prefix:  reversePrefix,
		},
	}

	conf := config.NXConfig{
		Namespaces: config.NamespaceConfig{
			DNS: config.DNSNamespaceConfig{
				Primaries: []config.PrimaryConfig{{
					Name:        "ns.primary.example",
					DottedEmail: `first\.last.ops.example.`,
					Zones:       []string{"example.com"},
				}},
			},
		},
	}
	defaultSOA := SOAInfo{
		NameserverFQDN:        "ns.default.test.",
		DottedMailResponsible: "hostmaster.default.test.",
		TTL:                   300,
		Refresh:               3600,
		Retry:                 600,
		Expire:                86400,
		BindDefaultRRTTL:      60,
		Serial:                "260915001",
	}

	zones := GenerateZones(addresses, defaultSOA, &conf)
	wantZones := []string{"100.51.198.in-addr.arpa", "alpha.test", "example.com"}
	slices.Sort(zones)
	if !slices.Equal(zones, wantZones) {
		t.Fatalf("GenerateZones() zones = %v, want %v", zones, wantZones)
	}

	indexPath := filepath.Join("generated", "search-index.json")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read search index: %v", err)
	}
	t.Logf("generated search index:\n%s", indexBytes)
	if len(indexBytes) == 0 || indexBytes[len(indexBytes)-1] != '\n' {
		t.Fatal("search index does not end with a newline")
	}
	if !strings.Contains(string(indexBytes), `"serial": 260915001`) {
		t.Fatalf("search index serial was not serialized as a number:\n%s", indexBytes)
	}
	if !strings.Contains(string(indexBytes), `"aliases": []`) {
		t.Fatalf("empty aliases were not serialized as an empty array:\n%s", indexBytes)
	}

	var got searchIndex
	if err := json.Unmarshal(indexBytes, &got); err != nil {
		t.Fatalf("unmarshal search index: %v", err)
	}
	if got.Version != 1 {
		t.Errorf("index version = %d, want 1", got.Version)
	}
	generatedAt, err := time.Parse(time.RFC3339, got.GeneratedAt)
	if err != nil {
		t.Errorf("generated_at = %q, want RFC3339: %v", got.GeneratedAt, err)
	} else if generatedAt.Location() != time.UTC {
		t.Errorf("generated_at location = %v, want UTC", generatedAt.Location())
	}

	wantIndex := searchIndex{
		Zones: []searchIndexZone{
			{Name: "alpha.test", Serial: 260915001, Nameserver: "ns.default.test", Contact: "hostmaster@default.test"},
			{Name: "example.com", Serial: 260915001, Nameserver: "ns.primary.example", Contact: "first.last@ops.example"},
		},
		Records: []searchIndexRecord{
			{Hostname: "alpha.alpha.test", Type: "AAAA", Value: "2001:db8::7", Aliases: []string{}, Prefix: "2001:db8::/64", Zone: "alpha.test"},
			{Hostname: "zzz.dev.example.com", Type: "A", Value: "192.0.2.42", Aliases: []string{"a-alias.example.com", "z-alias.example.com"}, Prefix: "192.0.2.0/24", Zone: "example.com"},
		},
	}
	if !reflect.DeepEqual(got.Zones, wantIndex.Zones) {
		t.Errorf("index zones = %#v, want %#v", got.Zones, wantIndex.Zones)
	}
	if !reflect.DeepEqual(got.Records, wantIndex.Records) {
		t.Errorf("index records = %#v, want %#v", got.Records, wantIndex.Records)
	}

	assertZoneLines(t, filepath.Join("generated", "zones", "example.com.db"),
		`@ IN SOA ns.primary.example. first\.last.ops.example. (`,
		`zzz.dev A 192.0.2.42`,
		`a-alias CNAME zzz.dev`,
		`z-alias CNAME zzz.dev`,
	)
	assertZoneLines(t, filepath.Join("generated", "zones", "alpha.test.db"),
		`@ IN SOA ns.default.test. hostmaster.default.test. (`,
		`alpha AAAA 2001:db8::7`,
	)
	assertZoneLines(t, filepath.Join("generated", "zones", "100.51.198.in-addr.arpa.db"),
		`9 PTR ptr-only.`,
	)

	conf.UpdatedFiles = nil
	defaultSOA.Serial = "260915999"
	secondZones := GenerateZones(addresses, defaultSOA, &conf)
	slices.Sort(secondZones)
	if !slices.Equal(secondZones, wantZones) {
		t.Errorf("unchanged GenerateZones() zones = %v, want %v", secondZones, wantZones)
	}
	if len(conf.UpdatedFiles) != 0 {
		t.Errorf("UpdatedFiles on unchanged run = %v, want none", conf.UpdatedFiles)
	}
	secondIndexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read unchanged search index: %v", err)
	}
	if !slices.Equal(indexBytes, secondIndexBytes) {
		t.Errorf("unchanged run rewrote search index\nbefore:\n%s\nafter:\n%s", indexBytes, secondIndexBytes)
	}
	assertZoneLines(t, filepath.Join("generated", "zones", "example.com.db"), `260915001 ; serial`)
	assertZoneLines(t, filepath.Join("generated", "zones", "alpha.test.db"), `260915001 ; serial`)
}

func assertZoneLines(t *testing.T, path string, expectedLines ...string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read zone %s: %v", path, err)
	}
	lines := make(map[string]bool)
	for line := range strings.SplitSeq(string(content), "\n") {
		lines[strings.Join(strings.Fields(line), " ")] = true
	}
	for _, expected := range expectedLines {
		if !lines[expected] {
			t.Errorf("zone %s does not contain normalized line %q:\n%s", path, expected, content)
		}
	}
}

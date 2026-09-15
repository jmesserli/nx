package dns

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSOAContact(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "regular", value: "hostmaster.example.com", want: "hostmaster@example.com"},
		{name: "application default", value: `unknown\.admin.local`, want: "unknown.admin@local"},
		{name: "escaped local dot actual", value: `first\.last.example.com`, want: "first.last@example.com"},
		{name: "trailing root dot", value: "hostmaster.example.com.", want: "hostmaster@example.com"},
		{name: "missing separator", value: `hostmaster\.example`, wantErr: true},
		{name: "empty local part", value: ".example.com", wantErr: true},
		{name: "empty domain", value: "hostmaster.", wantErr: true},
		{name: "empty domain label", value: "hostmaster.example..com", wantErr: true},
		{name: "invalid at sign", value: "host@master.example.com", wantErr: true},
		{name: "consecutive local dots", value: `first\..last.example.com`, wantErr: true},
		{name: "unsupported escape", value: `host\master.example.com`, wantErr: true},
		{name: "dangling domain escape", value: `hostmaster.example.com\`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := soaContact(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("soaContact(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("soaContact(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestNormalizeSearchIndex(t *testing.T) {
	index := searchIndex{
		Zones: []searchIndexZone{{Name: "z.example"}, {Name: "a.example"}},
		Records: []searchIndexRecord{
			{Hostname: "z.example", Type: "AAAA", Value: "::2", Aliases: []string{"z-alias", "a-alias"}},
			{Hostname: "a.example", Type: "AAAA", Value: "::1"},
			{Hostname: "a.example", Type: "A", Value: "192.0.2.2"},
			{Hostname: "a.example", Type: "A", Value: "192.0.2.1"},
		},
	}

	normalizeSearchIndex(&index)

	if got := []string{index.Zones[0].Name, index.Zones[1].Name}; !reflect.DeepEqual(got, []string{"a.example", "z.example"}) {
		t.Fatalf("zone order = %v", got)
	}
	if got := []string{index.Records[0].Value, index.Records[1].Value, index.Records[2].Type, index.Records[3].Hostname}; !reflect.DeepEqual(got, []string{"192.0.2.1", "192.0.2.2", "AAAA", "z.example"}) {
		t.Fatalf("record order = %v", got)
	}
	if !reflect.DeepEqual(index.Records[3].Aliases, []string{"a-alias", "z-alias"}) {
		t.Fatalf("aliases = %v", index.Records[3].Aliases)
	}
	if index.Records[0].Aliases == nil {
		t.Fatal("nil aliases were not normalized")
	}
}

func TestWriteSearchIndexCachesSemanticContent(t *testing.T) {
	t.Chdir(t.TempDir())

	index := searchIndex{
		Zones:   []searchIndexZone{{Name: "example.com", Serial: 260915001, Nameserver: "ns.example.com", Contact: "admin@example.com"}},
		Records: []searchIndexRecord{{Hostname: "host.example.com", Type: "A", Value: "192.0.2.1", Prefix: "192.0.2.0/24", Zone: "example.com"}},
	}
	written, err := writeSearchIndex(index)
	if err != nil || !written {
		t.Fatalf("first write = (%v, %v), want (true, nil)", written, err)
	}
	first, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(first), "\n") {
		t.Fatal("index does not end with a newline")
	}
	if !strings.Contains(string(first), `"aliases": []`) {
		t.Fatal("empty aliases were not serialized as []")
	}
	if strings.Contains(string(first), `"serial": "`) {
		t.Fatal("serial was serialized as a string")
	}

	written, err = writeSearchIndex(index)
	if err != nil || written {
		t.Fatalf("unchanged write = (%v, %v), want (false, nil)", written, err)
	}
	second, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("unchanged write modified the file")
	}

	if err := os.WriteFile(searchIndexPath, []byte("not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	written, err = writeSearchIndex(index)
	if err != nil || !written {
		t.Fatalf("malformed cache write = (%v, %v), want (true, nil)", written, err)
	}

	var oldVersion searchIndex
	content, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &oldVersion); err != nil {
		t.Fatal(err)
	}
	oldVersion.Version = 0
	content, err = json.Marshal(oldVersion)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(searchIndexPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	written, err = writeSearchIndex(index)
	if err != nil || !written {
		t.Fatalf("old version write = (%v, %v), want (true, nil)", written, err)
	}

	index.Records[0].Value = "192.0.2.2"
	written, err = writeSearchIndex(index)
	if err != nil || !written {
		t.Fatalf("changed write = (%v, %v), want (true, nil)", written, err)
	}

	var decoded searchIndex
	content, err = os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Version != searchIndexVersion || decoded.GeneratedAt == "" || decoded.Records[0].Value != "192.0.2.2" {
		t.Fatalf("unexpected index: %+v", decoded)
	}
}

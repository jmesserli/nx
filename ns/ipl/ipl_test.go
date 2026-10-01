package ipl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"peg.nu/nx/config"
	"peg.nu/nx/model"
	"peg.nu/nx/resolver"
)

func TestResolvedIPLists(t *testing.T) {
	templateContent, err := os.ReadFile(filepath.Join("..", "..", "templates", "ip-list.tmpl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, customFields := range []bool{false, true} {
		name := "legacy tags"
		if customFields {
			name = "custom fields"
		}
		t.Run(name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			if err := os.MkdirAll("templates", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll("generated/ipl", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile("templates/ip-list.tmpl", templateContent, 0o644); err != nil {
				t.Fatal(err)
			}
			prefix := model.IPAMPrefix{Tags: []model.Tag{{Name: "nx:ipl:enable[true]"}, {Name: "nx:ipl:list[internal]"}}}
			if customFields {
				enabled, lists := true, []string{"internal"}
				prefix.Tags = nil
				prefix.CustomFields = model.CustomFields{IPLEnabled: &enabled, IPLists: &lists}
			}
			resolver.ResolvePrefix(&prefix)
			addresses := []model.IPAddress{{Address: "192.0.2.10/24", Prefix: &prefix}, {Address: "2001:db8::10/64", Prefix: &prefix}}
			for _, body := range []string{
				`{"address":"192.0.2.11/24","custom_fields":{"nx_ip_lists_enabled":false}}`,
				`{"address":"192.0.2.12/24","custom_fields":{"nx_ip_lists":[]}}`,
				`{"address":"192.0.2.13/24","tags":[{"name":"nx:ipl:list[other]"}],"custom_fields":{"nx_ip_lists":["override"]}}`,
			} {
				var address model.IPAddress
				if err := json.Unmarshal([]byte(body), &address); err != nil {
					t.Fatal(err)
				}
				address.Prefix = &prefix
				addresses = append(addresses, address)
			}
			for i := range addresses {
				resolver.ResolveAddress(&addresses[i])
			}
			conf := config.NXConfig{}
			GenerateIPLists(addresses, &conf)
			body, err := os.ReadFile("generated/ipl/internal.ipl.txt")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "192.0.2.10") || !strings.Contains(string(body), "2001:db8::10") || strings.Contains(string(body), "192.0.2.11") || strings.Contains(string(body), "192.0.2.12") || strings.Contains(string(body), "192.0.2.13") {
				t.Fatalf("unexpected inherited list: %s", body)
			}
			body, err = os.ReadFile("generated/ipl/override.ipl.txt")
			if err != nil || !strings.Contains(string(body), "192.0.2.13") {
				t.Fatalf("missing override list: %s, %v", body, err)
			}
			if _, err := os.Stat("generated/ipl/other.ipl.txt"); !os.IsNotExist(err) {
				t.Fatal("legacy list survived custom override")
			}
			unchanged := config.NXConfig{}
			GenerateIPLists(addresses, &unchanged)
			if len(unchanged.UpdatedFiles) != 0 {
				t.Fatalf("unchanged lists rewritten: %v", unchanged.UpdatedFiles)
			}
		})
	}
}

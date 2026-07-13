package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"peg.nu/nx/config"
	"peg.nu/nx/model"
)

func TestChangedGeneratedFileReachesReports(t *testing.T) {
	templateFiles := []string{
		"bind-config.tmpl",
		"bind-zone.tmpl",
		"ip-list.tmpl",
		"wg-config.tmpl",
	}
	templateContents := make(map[string][]byte, len(templateFiles))
	for _, name := range templateFiles {
		content, err := os.ReadFile(filepath.Join("templates", name))
		if err != nil {
			t.Fatalf("read template %s: %v", name, err)
		}
		templateContents[name] = content
	}

	workingDirectory := t.TempDir()
	if err := os.Mkdir(filepath.Join(workingDirectory, "templates"), os.ModePerm); err != nil {
		t.Fatalf("create templates directory: %v", err)
	}
	for name, content := range templateContents {
		path := filepath.Join(workingDirectory, "templates", name)
		if err := os.WriteFile(path, content, os.ModePerm); err != nil {
			t.Fatalf("write template %s: %v", name, err)
		}
	}
	t.Chdir(workingDirectory)
	if err := prepareOutputDirectories("generated"); err != nil {
		t.Fatalf("prepareOutputDirectories() error = %v", err)
	}

	newConfig := func() config.NXConfig {
		return config.NXConfig{
			Namespaces: config.NamespaceConfig{
				DNS: config.DNSNamespaceConfig{
					Primaries: []config.PrimaryConfig{{
						Name: "ns1.example.com",
						IP:   "192.0.2.1",
						Zones: []string{
							"example.com",
						},
					}},
				},
			},
		}
	}
	newPrefixIPs := func(address string) []prefixIPs {
		prefix := model.IPAMPrefix{
			Prefix: "192.0.2.0/24",
			Tags: []model.Tag{
				{Name: "nx:dns:enable[true]"},
				{Name: "nx:dns:forward_zone[example.com]"},
			},
			EnOptions: model.EnableOptions{DNSEnabled: true},
		}
		return []prefixIPs{{
			prefix: prefix,
			ips: []model.IPAddress{{
				Address: address,
				DnsName: "host1",
				Prefix:  &prefix,
			}},
		}}
	}

	firstRun := newConfig()
	generateAll(newPrefixIPs("192.0.2.10/24"), nil, nil, nil, &firstRun)
	wantUpdatedFiles := []string{
		"generated/zones/example.com.db",
		"generated/bind-config/ns1.example.com.conf",
	}
	if strings.Join(firstRun.UpdatedFiles, "\n") != strings.Join(wantUpdatedFiles, "\n") {
		t.Fatalf("UpdatedFiles = %v, want %v", firstRun.UpdatedFiles, wantUpdatedFiles)
	}

	modifiedAt := time.Date(2026, time.July, 13, 10, 0, 0, 0, time.FixedZone("CEST", 2*60*60))
	if err := writeGenerationReports("generated", firstRun.UpdatedFiles, modifiedAt); err != nil {
		t.Fatalf("writeGenerationReports() error = %v", err)
	}
	assertFileContent(t, filepath.Join("generated", "updated_files.txt"), strings.Join(wantUpdatedFiles, "\n"))
	assertFileContent(t, filepath.Join("generated", "last_modified.txt"), modifiedAt.Format(time.RFC3339))

	secondRun := newConfig()
	generateAll(newPrefixIPs("192.0.2.10/24"), nil, nil, nil, &secondRun)
	if len(secondRun.UpdatedFiles) != 0 {
		t.Fatalf("UpdatedFiles on unchanged run = %v, want none", secondRun.UpdatedFiles)
	}

	thirdRun := newConfig()
	generateAll(newPrefixIPs("192.0.2.11/24"), nil, nil, nil, &thirdRun)
	wantUpdatedZone := "generated/zones/example.com.db"
	if len(thirdRun.UpdatedFiles) != 1 || thirdRun.UpdatedFiles[0] != wantUpdatedZone {
		t.Fatalf("UpdatedFiles after zone change = %v, want [%s]", thirdRun.UpdatedFiles, wantUpdatedZone)
	}

	zoneModifiedAt := modifiedAt.Add(time.Hour)
	if err := writeGenerationReports("generated", thirdRun.UpdatedFiles, zoneModifiedAt); err != nil {
		t.Fatalf("writeGenerationReports() after zone change error = %v", err)
	}
	assertFileContent(t, filepath.Join("generated", "updated_files.txt"), wantUpdatedZone)
	assertFileContent(t, filepath.Join("generated", "last_modified.txt"), zoneModifiedAt.Format(time.RFC3339))
}

func TestWriteGenerationReports(t *testing.T) {
	t.Run("writes last modified timestamp when a generated file changed", func(t *testing.T) {
		outputDirectory := t.TempDir()
		modifiedAt := time.Date(2026, time.July, 13, 9, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
		updatedFiles := []string{"generated/zones/example.com.db", "generated/ipl/internal.ipl.txt"}

		if err := writeGenerationReports(outputDirectory, updatedFiles, modifiedAt); err != nil {
			t.Fatalf("writeGenerationReports() error = %v", err)
		}

		assertFileContent(t, filepath.Join(outputDirectory, "updated_files.txt"), strings.Join(updatedFiles, "\n"))
		assertFileContent(t, filepath.Join(outputDirectory, "last_modified.txt"), modifiedAt.Format(time.RFC3339))
	})

	t.Run("preserves last modified timestamp when no generated file changed", func(t *testing.T) {
		outputDirectory := t.TempDir()
		lastModifiedPath := filepath.Join(outputDirectory, "last_modified.txt")
		previousTimestamp := "2026-07-12T09:30:00+02:00"
		if err := os.WriteFile(lastModifiedPath, []byte(previousTimestamp), os.ModePerm); err != nil {
			t.Fatalf("write test fixture: %v", err)
		}

		if err := writeGenerationReports(outputDirectory, nil, time.Now()); err != nil {
			t.Fatalf("writeGenerationReports() error = %v", err)
		}

		assertFileContent(t, filepath.Join(outputDirectory, "updated_files.txt"), "")
		assertFileContent(t, lastModifiedPath, previousTimestamp)
	})

	t.Run("returns an error when last modified report cannot be written", func(t *testing.T) {
		outputDirectory := t.TempDir()
		lastModifiedPath := filepath.Join(outputDirectory, "last_modified.txt")
		if err := os.Mkdir(lastModifiedPath, os.ModePerm); err != nil {
			t.Fatalf("create conflicting directory: %v", err)
		}

		err := writeGenerationReports(outputDirectory, []string{"generated/zones/example.com.db"}, time.Now())
		if err == nil {
			t.Fatal("writeGenerationReports() error = nil, want an error")
		}
		if !strings.Contains(err.Error(), "write last modified report") {
			t.Fatalf("writeGenerationReports() error = %q, want last modified context", err)
		}
	})
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(content) != expected {
		t.Errorf("content of %s = %q, want %q", path, content, expected)
	}
}

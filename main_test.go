package main

import (
	"bytes"
	"encoding/json"
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
						Name:        "ns1.example.com",
						IP:          "192.0.2.1",
						DottedEmail: "hostmaster.example.com",
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
	searchIndexPath := filepath.Join("generated", "search-index.json")
	wantUpdatedFiles := []string{
		"generated/zones/example.com.db",
		"generated/search-index.json",
		"generated/search-index.json.gz",
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

	firstIndexBytes, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatalf("read generated search index: %v", err)
	}
	assertSearchIndex(t, firstIndexBytes, "192.0.2.10")

	indexSentinelTime := time.Date(2001, time.February, 3, 4, 5, 6, 0, time.UTC)
	if err := os.Chtimes(searchIndexPath, indexSentinelTime, indexSentinelTime); err != nil {
		t.Fatalf("set search index sentinel timestamp: %v", err)
	}
	indexInfoBeforeSecondRun, err := os.Stat(searchIndexPath)
	if err != nil {
		t.Fatalf("stat search index before unchanged run: %v", err)
	}

	secondRun := newConfig()
	generateAll(newPrefixIPs("192.0.2.10/24"), nil, nil, nil, &secondRun)
	if len(secondRun.UpdatedFiles) != 0 {
		t.Fatalf("UpdatedFiles on unchanged run = %v, want none", secondRun.UpdatedFiles)
	}
	secondIndexBytes, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatalf("read search index after unchanged run: %v", err)
	}
	if !bytes.Equal(secondIndexBytes, firstIndexBytes) {
		t.Error("search index bytes changed on unchanged run")
	}
	indexInfoAfterSecondRun, err := os.Stat(searchIndexPath)
	if err != nil {
		t.Fatalf("stat search index after unchanged run: %v", err)
	}
	if !indexInfoAfterSecondRun.ModTime().Equal(indexInfoBeforeSecondRun.ModTime()) {
		t.Errorf("search index modification time = %s, want unchanged %s", indexInfoAfterSecondRun.ModTime(), indexInfoBeforeSecondRun.ModTime())
	}

	unchangedRunTime := modifiedAt.Add(30 * time.Minute)
	if err := writeGenerationReports("generated", secondRun.UpdatedFiles, unchangedRunTime); err != nil {
		t.Fatalf("writeGenerationReports() after unchanged run error = %v", err)
	}
	assertFileContent(t, filepath.Join("generated", "updated_files.txt"), "")
	assertFileContent(t, filepath.Join("generated", "last_modified.txt"), modifiedAt.Format(time.RFC3339))

	thirdRun := newConfig()
	generateAll(newPrefixIPs("192.0.2.11/24"), nil, nil, nil, &thirdRun)
	wantChangedFiles := []string{
		"generated/zones/example.com.db",
		"generated/search-index.json",
		"generated/search-index.json.gz",
	}
	if strings.Join(thirdRun.UpdatedFiles, "\n") != strings.Join(wantChangedFiles, "\n") {
		t.Fatalf("UpdatedFiles after zone change = %v, want %v", thirdRun.UpdatedFiles, wantChangedFiles)
	}
	thirdIndexBytes, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatalf("read search index after zone change: %v", err)
	}
	assertSearchIndex(t, thirdIndexBytes, "192.0.2.11")

	zoneModifiedAt := modifiedAt.Add(time.Hour)
	if err := writeGenerationReports("generated", thirdRun.UpdatedFiles, zoneModifiedAt); err != nil {
		t.Fatalf("writeGenerationReports() after zone change error = %v", err)
	}
	assertFileContent(t, filepath.Join("generated", "updated_files.txt"), strings.Join(wantChangedFiles, "\n"))
	assertFileContent(t, filepath.Join("generated", "last_modified.txt"), zoneModifiedAt.Format(time.RFC3339))

	invalidVersionIndex := bytes.Replace(thirdIndexBytes, []byte(`"version": 1`), []byte(`"version": 0`), 1)
	if bytes.Equal(invalidVersionIndex, thirdIndexBytes) {
		t.Fatal("search index fixture does not contain version 1")
	}
	if err := os.WriteFile(searchIndexPath, invalidVersionIndex, os.ModePerm); err != nil {
		t.Fatalf("write search index with invalid version: %v", err)
	}

	fourthRun := newConfig()
	generateAll(newPrefixIPs("192.0.2.11/24"), nil, nil, nil, &fourthRun)
	wantUpdatedIndexes := []string{"generated/search-index.json", "generated/search-index.json.gz"}
	if strings.Join(fourthRun.UpdatedFiles, "\n") != strings.Join(wantUpdatedIndexes, "\n") {
		t.Fatalf("UpdatedFiles after index version change = %v, want %v", fourthRun.UpdatedFiles, wantUpdatedIndexes)
	}
	repairedIndexBytes, err := os.ReadFile(searchIndexPath)
	if err != nil {
		t.Fatalf("read repaired search index: %v", err)
	}
	assertSearchIndex(t, repairedIndexBytes, "192.0.2.11")

	indexModifiedAt := zoneModifiedAt.Add(time.Hour)
	if err := writeGenerationReports("generated", fourthRun.UpdatedFiles, indexModifiedAt); err != nil {
		t.Fatalf("writeGenerationReports() after index-only change error = %v", err)
	}
	assertFileContent(t, filepath.Join("generated", "updated_files.txt"), strings.Join(wantUpdatedIndexes, "\n"))
	assertFileContent(t, filepath.Join("generated", "last_modified.txt"), indexModifiedAt.Format(time.RFC3339))
}

func assertSearchIndex(t *testing.T, content []byte, wantAddress string) {
	t.Helper()

	var index struct {
		Version     int    `json:"version"`
		GeneratedAt string `json:"generated_at"`
		Zones       []struct {
			Name       string `json:"name"`
			Serial     int    `json:"serial"`
			Nameserver string `json:"nameserver"`
			Contact    string `json:"contact"`
		} `json:"zones"`
		Records []struct {
			Hostname string   `json:"hostname"`
			Type     string   `json:"type"`
			Value    string   `json:"value"`
			Aliases  []string `json:"aliases"`
			Prefix   string   `json:"prefix"`
			Zone     string   `json:"zone"`
		} `json:"records"`
	}
	if err := json.Unmarshal(content, &index); err != nil {
		t.Fatalf("unmarshal generated search index: %v", err)
	}
	if index.Version != 1 {
		t.Errorf("search index version = %d, want 1", index.Version)
	}
	generatedAt, err := time.Parse(time.RFC3339, index.GeneratedAt)
	if err != nil {
		t.Fatalf("search index generated_at = %q, want RFC3339 timestamp: %v", index.GeneratedAt, err)
	}
	_, offset := generatedAt.Zone()
	if offset != 0 || !strings.HasSuffix(index.GeneratedAt, "Z") {
		t.Errorf("search index generated_at = %q, want UTC timestamp ending in Z", index.GeneratedAt)
	}
	if len(index.Zones) != 1 {
		t.Fatalf("search index zones = %+v, want one zone", index.Zones)
	}
	zone := index.Zones[0]
	if zone.Name != "example.com" || zone.Serial <= 0 || zone.Nameserver != "ns1.example.com" || zone.Contact != "hostmaster@example.com" {
		t.Errorf("search index zone = %+v, want example.com with numeric serial, ns1.example.com and hostmaster@example.com", zone)
	}
	if len(index.Records) != 1 {
		t.Fatalf("search index records = %+v, want one record", index.Records)
	}
	record := index.Records[0]
	if record.Hostname != "host1.example.com" || record.Type != "A" || record.Value != wantAddress || record.Prefix != "192.0.2.0/24" || record.Zone != "example.com" {
		t.Errorf("search index record = %+v, want host1.example.com A %s in example.com with prefix 192.0.2.0/24", record, wantAddress)
	}
	if record.Aliases == nil || len(record.Aliases) != 0 {
		t.Errorf("search index aliases = %#v, want non-nil empty list", record.Aliases)
	}
	if len(content) == 0 || content[len(content)-1] != '\n' {
		t.Error("generated search index does not end with a newline")
	}
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

package dns

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"
)

const (
	searchIndexVersion  = 1
	searchIndexPath     = "generated/search-index.json"
	searchIndexGzipPath = searchIndexPath + ".gz"
)

// searchIndex describes the generated DNS search document. These types are
// deliberately private because they are the representation of one output
// format, rather than part of the DNS model used by the generators.
type searchIndex struct {
	Version     int                 `json:"version"`
	GeneratedAt string              `json:"generated_at"`
	Zones       []searchIndexZone   `json:"zones"`
	Records     []searchIndexRecord `json:"records"`
}

type searchIndexZone struct {
	Name       string `json:"name"`
	Serial     uint32 `json:"serial"`
	Nameserver string `json:"nameserver"`
	Contact    string `json:"contact"`
}

type searchIndexRecord struct {
	Hostname string   `json:"hostname"`
	Type     string   `json:"type"`
	Value    string   `json:"value"`
	Aliases  []string `json:"aliases"`
	Prefix   string   `json:"prefix"`
	Zone     string   `json:"zone"`
}

// normalizeSearchIndex makes the output stable even when its input was
// collected by iterating over maps.
func normalizeSearchIndex(index *searchIndex) {
	if index.Zones == nil {
		index.Zones = []searchIndexZone{}
	}
	if index.Records == nil {
		index.Records = []searchIndexRecord{}
	}

	for i := range index.Records {
		if index.Records[i].Aliases == nil {
			index.Records[i].Aliases = []string{}
		}
		slices.Sort(index.Records[i].Aliases)
	}

	slices.SortFunc(index.Zones, func(a, b searchIndexZone) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(index.Records, func(a, b searchIndexRecord) int {
		if result := strings.Compare(a.Hostname, b.Hostname); result != 0 {
			return result
		}
		if result := strings.Compare(a.Type, b.Type); result != 0 {
			return result
		}
		if result := strings.Compare(a.Value, b.Value); result != 0 {
			return result
		}
		if result := strings.Compare(a.Zone, b.Zone); result != 0 {
			return result
		}
		if result := strings.Compare(a.Prefix, b.Prefix); result != 0 {
			return result
		}
		return slices.Compare(a.Aliases, b.Aliases)
	})
}

// soaContact converts the DNS representation of an SOA RNAME into an email
// address. The first unescaped dot separates its local and domain parts.
func soaContact(value string) (string, error) {
	separator := -1
	for i := 0; i < len(value); i++ {
		if value[i] == '\\' {
			if i+1 >= len(value) || value[i+1] != '.' {
				return "", fmt.Errorf("unsupported escape in SOA mailbox %q", value)
			}
			i++
			continue
		}
		if value[i] == '.' {
			separator = i
			break
		}
	}

	if separator <= 0 || separator == len(value)-1 {
		return "", fmt.Errorf("malformed SOA mailbox %q", value)
	}

	local := strings.ReplaceAll(value[:separator], `\.`, ".")
	domain := strings.TrimSuffix(value[separator+1:], ".")
	if !validSOAMailLocal(local) || domain == "" || !validSOAMailDomain(domain) {
		return "", fmt.Errorf("malformed SOA mailbox %q", value)
	}
	return local + "@" + domain, nil
}

func validSOAMailLocal(local string) bool {
	if local == "" || strings.HasPrefix(local, ".") || strings.HasSuffix(local, ".") || strings.Contains(local, "..") {
		return false
	}
	const specials = "!#$%&'*+-/=?^_`{|}~."
	for _, char := range local {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') && !strings.ContainsRune(specials, char) {
			return false
		}
	}
	return true
}

func validSOAMailDomain(domain string) bool {
	if len(domain) > 253 || strings.ContainsAny(domain, "\\@ \t\r\n") {
		return false
	}
	for label := range strings.SplitSeq(domain, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') &&
				(char < '0' || char > '9') && char != '-' {
				return false
			}
		}
	}
	return true
}

// writeSearchIndex writes the JSON index only when its semantic DNS content has
// changed. The generated timestamp is intentionally excluded from that
// comparison. Its gzip companion is repaired independently without touching an
// otherwise-current JSON file.
func writeSearchIndex(index searchIndex) ([]string, error) {
	index.Version = searchIndexVersion
	normalizeSearchIndex(&index)

	var existing searchIndex
	if content, err := os.ReadFile(searchIndexPath); err == nil {
		if err := json.Unmarshal(content, &existing); err == nil &&
			existing.Version == index.Version &&
			reflect.DeepEqual(existing.Zones, index.Zones) &&
			reflect.DeepEqual(existing.Records, index.Records) {
			gzipCurrent, err := gzipContains(searchIndexGzipPath, content)
			if err != nil {
				return nil, err
			}
			if gzipCurrent {
				return nil, nil
			}
			if err := writeGzip(searchIndexGzipPath, content); err != nil {
				return nil, err
			}
			return []string{searchIndexGzipPath}, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	index.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	content, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, err
	}
	content = append(content, '\n')

	if err := os.MkdirAll(filepath.Dir(searchIndexPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(searchIndexPath, content, 0o644); err != nil {
		return nil, err
	}
	if err := writeGzip(searchIndexGzipPath, content); err != nil {
		return nil, err
	}
	return []string{searchIndexPath, searchIndexGzipPath}, nil
}

func gzipContains(path string, want []byte) (bool, error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer file.Close()

	reader, err := gzip.NewReader(file)
	if err != nil {
		return false, nil
	}
	content, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		return false, nil
	}
	return bytes.Equal(content, want), nil
}

func writeGzip(path string, content []byte) error {
	var compressed bytes.Buffer
	writer, err := gzip.NewWriterLevel(&compressed, gzip.BestCompression)
	if err != nil {
		return err
	}
	if _, err := writer.Write(content); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, compressed.Bytes(), 0o644)
}

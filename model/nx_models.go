package model

// Configuration contains effective settings, independent of their NetBox source.
type Configuration struct {
	DNSEnabled         bool
	DNSForwardDisabled bool
	DNSForwardZone     string
	DNSReverseZone     string
	DNSCNames          []string
	IPLEnabled         bool
	IPLists            []string
}

// CustomFields uses nil for missing/null values and pointers for explicit overrides.
type CustomFields struct {
	DNSEnabled         *bool     `json:"nx_dns_enabled"`
	DNSForwardDisabled *bool     `json:"nx_dns_forward_disabled"`
	DNSForwardZone     *string   `json:"nx_dns_forward_zone"`
	DNSReverseZone     *string   `json:"nx_dns_reverse_zone"`
	DNSCNames          *[]string `json:"nx_dns_cnames"`
	IPLEnabled         *bool     `json:"nx_ip_lists_enabled"`
	IPLists            *[]string `json:"nx_ip_lists"`
}

type IPAMPrefix struct {
	ID     int    `json:"id"`
	Prefix string `json:"prefix"`

	CustomFields CustomFields  `json:"custom_fields"`
	Config       Configuration `json:"-"`
}

type IPAddress struct {
	ID           int    `json:"id"`
	Address      string `json:"address"`
	DnsName      string `json:"dns_name"`
	Description  string `json:"description"`
	Prefix       *IPAMPrefix
	CustomFields CustomFields  `json:"custom_fields"`
	Config       Configuration `json:"-"`
}

func (i IPAddress) GetName() string {
	if i.DnsName == "" {
		return i.Description
	} else {
		return i.DnsName
	}
}

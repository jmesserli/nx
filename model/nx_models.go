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
// The nx annotations are only used by the temporary legacy-tag adapter.
type CustomFields struct {
	DNSEnabled         *bool     `json:"nx_dns_enabled" nx:"enable,ns:dns"`
	DNSForwardDisabled *bool     `json:"nx_dns_forward_disabled"`
	DNSForwardZone     *string   `json:"nx_dns_forward_zone" nx:"forward_zone,ns:dns"`
	DNSReverseZone     *string   `json:"nx_dns_reverse_zone" nx:"reverse_zone,ns:dns"`
	DNSCNames          *[]string `json:"nx_dns_cnames" nx:"cname,ns:dns"`
	IPLEnabled         *bool     `json:"nx_ip_lists_enabled" nx:"enable,ns:ipl"`
	IPLists            *[]string `json:"nx_ip_lists" nx:"list,ns:ipl"`
}

type IPAMPrefix struct {
	ID     int    `json:"id"`
	Prefix string `json:"prefix"`
	Tags   []Tag  `json:"tags"`

	CustomFields CustomFields  `json:"custom_fields"`
	Config       Configuration `json:"-"`
}

type Tag struct {
	ID    int    `json:"id"`
	URL   string `json:"url"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

type IPAddress struct {
	ID           int    `json:"id"`
	Address      string `json:"address"`
	DnsName      string `json:"dns_name"`
	Description  string `json:"description"`
	Tags         []Tag  `json:"tags"`
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

package model

type EnableOptions struct {
	DNSEnabled bool   `nx:"enable,ns:dns"`
	WGVpnName  string `nx:"mesh,ns:wg"`
	IPLEnabled bool   `nx:"enable,ns:ipl"`
}

type IPAMPrefix struct {
	ID     int    `json:"id"`
	Prefix string `json:"prefix"`
	Tags   []Tag  `json:"tags"`

	EnOptions EnableOptions
}

type Tag struct {
	ID    int    `json:"id"`
	URL   string `json:"url"`
	Name  string `json:"name"`
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

type IPAddress struct {
	ID          int    `json:"id"`
	Address     string `json:"address"`
	DnsName     string `json:"dns_name"`
	Description string `json:"description"`
	Tags        []Tag  `json:"tags"`
	Prefix      *IPAMPrefix
}

func (i IPAddress) GetName() string {
	if i.DnsName == "" {
		return i.Description
	} else {
		return i.DnsName
	}
}

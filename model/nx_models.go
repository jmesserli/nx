package model

type Configuration struct {
	DnsEnabled     bool     `nx:"enable,ns:dns"`
	DnsForwardZone string   `nx:"forward_zone,ns:dns"`
	DnsReverseZone string   `nx:"reverse_zone,ns:dns"`
	DnsCNames      []string `nx:"cname,ns:dns"`

	IpListsEnabled bool     `nx:"enable,ns:ipl"`
	IpLists        []string `nx:"list,ns:ipl"`
}

type IPAMPrefix struct {
	Prefix string `json:"prefix"`
	Tags   []Tag  `json:"tags"`

	Config Configuration
}

type Tag struct {
	Name string `json:"name"`
}

func (t Tag) GetName() string {
	return t.Name
}

type IPAddress struct {
	ID          int    `json:"id"`
	Address     string `json:"address"`
	DnsName     string `json:"dns_name"`
	Description string `json:"description"`
	Tags        []Tag  `json:"tags"`
	Prefix      *IPAMPrefix
	Config      Configuration
}

func (i IPAddress) GetName() string {
	if i.DnsName == "" {
		return i.Description
	} else {
		return i.DnsName
	}
}

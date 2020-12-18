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
	ID      int    `json:"id"`
	Address string `json:"address"`
	Name    string `json:"description"`
	Tags    []Tag  `json:"tags"`
	Prefix  *IPAMPrefix
}

package netbox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"peg.nu/nx/config"
	"peg.nu/nx/tagparser"
)

type Client struct {
	conf config.NXConfig
}

func New(conf config.NXConfig) Client {
	return Client{
		conf: conf,
	}
}

func (c Client) performGET(path string, query string) []byte {
	if len(query) == 0 {
		query = "?limit=2000"
	} else {
		query = query + "&limit=2000"
	}

	url := fmt.Sprintf("%v%v%v", c.conf.Netbox.URL, path, query)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Token %v", c.conf.Netbox.ApiKey))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	return body
}

type ipamPrefixResponse struct {
	Count   int          `json:"count"`
	Results []IPAMPrefix `json:"results"`
}

type EnableOptions struct {
	DNSEnabled bool   `nx:"enable,ns:dns"`
	WGVpnName  string `nx:"mesh,ns:wg"`
}

type IPAMPrefix struct {
	ID     int      `json:"id"`
	Prefix string   `json:"prefix"`
	Tags   []string `json:"tags"`

	EnOptions EnableOptions
}

func (c Client) GetIPAMPrefixes() []IPAMPrefix {
	response := ipamPrefixResponse{}
	json.Unmarshal(c.performGET("/ipam/prefixes/", ""), &response)

	for i := range response.Results {
		prefix := &response.Results[i]
		prefix.EnOptions = EnableOptions{}

		tagparser.ParseTags(&prefix.EnOptions, prefix.Tags, []string{})
	}

	return response.Results
}

type IPAddress struct {
	ID      int      `json:"id"`
	Address string   `json:"address"`
	Name    string   `json:"description"`
	Tags    []string `json:"tags"`
	Prefix  *IPAMPrefix
}

type ipAddressResponse struct {
	Count   int         `json:"count"`
	Results []IPAddress `json:"results"`
}

func (c Client) GetIPAddressesByPrefix(prefix IPAMPrefix) []IPAddress {
	response := ipAddressResponse{}
	json.Unmarshal(c.performGET("/ipam/ip-addresses/", fmt.Sprintf("?parent=%s", url.QueryEscape(prefix.Prefix))), &response)

	for i := range response.Results {
		ip := &response.Results[i]
		ip.Prefix = &prefix
	}

	return response.Results
}

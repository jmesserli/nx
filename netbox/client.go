package netbox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/jmesserli/nx/config"
	"github.com/jmesserli/nx/tagparser"
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

type GenerateOptions struct {
	Enabled bool `nx:"enable,ns:dns"`

	ReverseZoneName string `nx:"reverse_zone,ns:dns"`
	ForwardZoneName string `nx:"forward_zone,ns:dns"`

	CNames []string `nx:"cname,ns:dns"`
}

type IPAMPrefix struct {
	ID     int      `json:"id"`
	Prefix string   `json:"prefix"`
	Tags   []string `json:"tags"`

	GenOptions GenerateOptions
}

func parseGenerateOptions(tags []string, parentPrefix *IPAMPrefix, options *GenerateOptions) {
	pTags := []string{}
	if parentPrefix != nil {
		pTags = parentPrefix.Tags
	}
	tagparser.ParseTags(options, tags, pTags)
}

func (c Client) GetIPAMPrefixes() []IPAMPrefix {
	response := ipamPrefixResponse{}
	json.Unmarshal(c.performGET("/ipam/prefixes/", ""), &response)

	for i := range response.Results {
		prefix := &response.Results[i]
		prefix.GenOptions = GenerateOptions{}

		parseGenerateOptions(prefix.Tags, nil, &prefix.GenOptions)
	}

	return response.Results
}

type IPAddress struct {
	ID      int      `json:"id"`
	Address string   `json:"address"`
	Name    string   `json:"description"`
	Tags    []string `json:"tags"`

	GenOptions GenerateOptions
}

type ipAddressResponse struct {
	Count   int         `json:"count"`
	Results []IPAddress `json:"results"`
}

func (c Client) GetIPAddressesByPrefix(prefix *IPAMPrefix) []IPAddress {
	response := ipAddressResponse{}
	json.Unmarshal(c.performGET("/ipam/ip-addresses/", fmt.Sprintf("?parent=%s", url.QueryEscape(prefix.Prefix))), &response)

	for i := range response.Results {
		address := &response.Results[i]
		address.GenOptions = GenerateOptions{}

		parseGenerateOptions(address.Tags, prefix, &address.GenOptions)
	}

	return response.Results
}

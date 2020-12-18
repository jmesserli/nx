package netbox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"peg.nu/nx/model"

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
	if res.StatusCode != http.StatusOK {
		log.Fatal(fmt.Errorf("netbox returned a non 200 status code: %s", res.Status))
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	return body
}

type ipamPrefixResponse struct {
	Count   int                `json:"count"`
	Results []model.IPAMPrefix `json:"results"`
}

func (c Client) GetIPAMPrefixes() []model.IPAMPrefix {
	response := ipamPrefixResponse{}
	json.Unmarshal(c.performGET("/ipam/prefixes/", ""), &response)

	for i := range response.Results {
		prefix := &response.Results[i]
		prefix.EnOptions = model.EnableOptions{}

		tagparser.ParseTags(&prefix.EnOptions, prefix.Tags, []model.Tag{})
	}

	return response.Results
}

type ipAddressResponse struct {
	Count   int               `json:"count"`
	Results []model.IPAddress `json:"results"`
}

func (c Client) GetIPAddressesByPrefix(prefix model.IPAMPrefix) []model.IPAddress {
	response := ipAddressResponse{}
	json.Unmarshal(c.performGET("/ipam/ip-addresses/", fmt.Sprintf("?parent=%s", url.QueryEscape(prefix.Prefix))), &response)

	for i := range response.Results {
		ip := &response.Results[i]
		ip.Prefix = &prefix
	}

	return response.Results
}

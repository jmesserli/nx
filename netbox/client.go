package netbox

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"peg.nu/nx/model"
	"strings"

	"peg.nu/nx/config"
	"peg.nu/nx/tagparser"
)

type Client struct {
	conf   config.NXConfig
	logger *log.Logger
}

func New(conf config.NXConfig) Client {
	return Client{
		conf:   conf,
		logger: log.New(os.Stdout, "[client] ", log.LstdFlags),
	}
}

func (c Client) performGET(path string, requestBody *string) []byte {
	var requestBodyReader = strings.NewReader("")
	if requestBody != nil {
		requestBodyReader = strings.NewReader(*requestBody)
	}

	requestUrl := fmt.Sprintf("%v%v%v", c.conf.Netbox.URL, path, nil)
	req, _ := http.NewRequest("GET", requestUrl, requestBodyReader)
	req.Header.Add("Authorization", fmt.Sprintf("Token %v", c.conf.Netbox.ApiKey))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	if res.StatusCode != http.StatusOK {
		log.Fatal(fmt.Errorf("netbox returned a non 200 status code: %s", res.Status))
	}

	defer func(closeable io.Closer) {
		err := closeable.Close()
		if err != nil {
			panic(err)
		}
	}(res.Body)

	responseBody, _ := io.ReadAll(res.Body)
	responseBodyStr := string(responseBody)
	if strings.TrimSpace(responseBodyStr) == "" {
		log.Fatal("Empty body returned")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		log.Println(fmt.Sprintf("Error Response: %d - body: \n%s", res.StatusCode, responseBodyStr))
	}

	return responseBody
}

type graphqlResponse[T interface{}] struct {
	Data T `json:"data"`
}

type graphqlPrefixList struct {
	PrefixList []graphqlPrefix `json:"prefix_list"`
}

type graphqlPrefix struct {
	Prefix       string              `json:"prefix"`
	Tags         []graphqlTag        `json:"tags"`
	CustomFields graphqlCustomFields `json:"custom_fields"`
}

type graphqlTag struct {
	Name string `json:"name"`
}

func (t graphqlTag) GetName() string {
	return t.Name
}

type graphqlCustomFields struct {
	DnsEnabled     *bool     `json:"nx_dns_enabled"`
	DnsForwardZone *string   `json:"nx_dns_forward_zone"`
	DnsReverseZone *string   `json:"nx_dns_reverse_zone"`
	DnsCNames      *[]string `json:"nx_dns_cnames"`

	IpListsEnabled *bool     `json:"nx_ip_lists_enabled"`
	IpLists        *[]string `json:"nx_ip_lists"`
}

func (c Client) GetIPAMPrefixes() []model.IPAMPrefix {
	var graphQlQuery = "query {\n  prefix_list {\n  prefix\n  tags {\n  name\n}\ncustom_fields\n}\n}\n"

	res := graphqlResponse[graphqlPrefixList]{}
	err := json.Unmarshal(c.performGET("/graphql/", &graphQlQuery), &res)
	if err != nil {
		panic(err)
	}

	var prefixes []model.IPAMPrefix
	for i := range res.Data.PrefixList {
		gqlPrefix := res.Data.PrefixList[i]

		modelConfig := model.Configuration{}

		var modelTags []model.Tag
		for _, tag := range gqlPrefix.Tags {
			modelTags = append(modelTags, model.Tag{Name: tag.Name})
		}

		// Then fully migrated, remove this line
		tagparser.ParseTags(&modelConfig, modelTags, []model.Tag{})

		// Custom fields take precedence
		if gqlPrefix.CustomFields.DnsEnabled != nil {
			modelConfig.DnsEnabled = *gqlPrefix.CustomFields.DnsEnabled
		}
		if gqlPrefix.CustomFields.DnsForwardZone != nil {
			modelConfig.DnsForwardZone = *gqlPrefix.CustomFields.DnsForwardZone
		}
		if gqlPrefix.CustomFields.DnsReverseZone != nil {
			modelConfig.DnsReverseZone = *gqlPrefix.CustomFields.DnsReverseZone
		}
		if gqlPrefix.CustomFields.DnsCNames != nil {
			modelConfig.DnsCNames = *gqlPrefix.CustomFields.DnsCNames
		}
		if gqlPrefix.CustomFields.IpListsEnabled != nil {
			modelConfig.IpListsEnabled = *gqlPrefix.CustomFields.IpListsEnabled
		}
		if gqlPrefix.CustomFields.IpLists != nil {
			modelConfig.IpLists = *gqlPrefix.CustomFields.IpLists
		}

		modelPrefix := model.IPAMPrefix{}
		modelPrefix.Prefix = gqlPrefix.Prefix
		modelPrefix.Config = modelConfig
		modelPrefix.Tags = modelTags

		prefixes = append(prefixes, modelPrefix)
	}

	return prefixes
}

type ipAddressResponse struct {
	Count   int               `json:"count"`
	Results []model.IPAddress `json:"results"`
}

func (c Client) GetIPAddresses(prefixes []model.IPAMPrefix) []model.IPAddress {
	response := ipAddressResponse{}
	err := json.Unmarshal(c.performGET("/ipam/ip-addresses/", fmt.Sprintf("?parent=%s", url.QueryEscape(prefix.Prefix))), &response)
	if err != nil {
		panic(err)
	}

	for i := range response.Results {
		ip := &response.Results[i]
		ip.Prefix = &prefix
	}

	return response.Results
}

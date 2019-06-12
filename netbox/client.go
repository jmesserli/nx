package netbox

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"

	"github.com/jmesserli/netbox-to-bind/util"
)

const baseURL = "https://netbox.pegnu.cloud/api"
const apiKey = "a9ed5b567a4b7de4e9e40ec15bb60edf993b966e"

func performGET(path string, query string) []byte {
	if len(query) == 0 {
		query = "?limit=2000"
	} else {
		query = query + "&limit=2000"
	}

	url := fmt.Sprintf("%v%v%v", baseURL, path, query)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Token %v", apiKey))
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
	Enabled bool

	ReverseEnabled bool
	ForwardEnabled bool

	ReverseZoneName string
	ForwardZoneName string

	CNames []string
}

type IPAMPrefix struct {
	ID     int      `json:"id"`
	Prefix string   `json:"prefix"`
	Tags   []string `json:"tags"`

	GenOptions GenerateOptions
}

var enabledRegex = regexp.MustCompile("nbbx_enabled?:(true|false)")
var reverseEnabledRegex = regexp.MustCompile("nbbx_reverse(?:_enabled?)?:(true|false)")
var forwardEnabledRegex = regexp.MustCompile("nbbx_forward(?:_enabled?)?:(true|false)")
var forwardZoneRegex = regexp.MustCompile("nbbx_forward_zone:(.+)")
var reverseZoneRegex = regexp.MustCompile("nbbx_reverse_zone:(.+)")
var cnameRegex = regexp.MustCompile("nbbx_cname:(.+)")

func parseGenerateOptions(tags []string, parentPrefix *IPAMPrefix, options *GenerateOptions) {
	if parentPrefix != nil {
		pOptions := parentPrefix.GenOptions

		options.Enabled = pOptions.Enabled
		options.ForwardEnabled = pOptions.ForwardEnabled
		options.ReverseEnabled = pOptions.ReverseEnabled
		options.ForwardZoneName = pOptions.ForwardZoneName
		options.ReverseZoneName = pOptions.ReverseZoneName
	}
	options.CNames = []string{}

	for _, tag := range tags {
		matches := enabledRegex.FindStringSubmatch(tag)
		if matches != nil {
			options.Enabled = util.MustConvertToBool(matches[1])
			continue
		}

		matches = reverseEnabledRegex.FindStringSubmatch(tag)
		if matches != nil {
			options.ReverseEnabled = util.MustConvertToBool(matches[1])
			continue
		}

		matches = forwardEnabledRegex.FindStringSubmatch(tag)
		if matches != nil {
			options.ForwardEnabled = util.MustConvertToBool(matches[1])
			continue
		}

		matches = reverseZoneRegex.FindStringSubmatch(tag)
		if matches != nil {
			options.ReverseZoneName = matches[1]
			continue
		}

		matches = forwardZoneRegex.FindStringSubmatch(tag)
		if matches != nil {
			options.ForwardZoneName = matches[1]
			continue
		}

		matches = cnameRegex.FindStringSubmatch(tag)
		if matches != nil {
			options.CNames = append(options.CNames, matches[1])
		}
	}
}

func GetIPAMPrefixes() []IPAMPrefix {
	response := ipamPrefixResponse{}
	json.Unmarshal(performGET("/ipam/prefixes/", ""), &response)

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

func GetIPAddressesByPrefix(prefix *IPAMPrefix) []IPAddress {
	response := ipAddressResponse{}
	json.Unmarshal(performGET("/ipam/ip-addresses/", fmt.Sprintf("?parent=%s", url.QueryEscape(prefix.Prefix))), &response)

	for i := range response.Results {
		address := &response.Results[i]
		address.GenOptions = GenerateOptions{}

		parseGenerateOptions(address.Tags, prefix, &address.GenOptions)
	}

	return response.Results
}

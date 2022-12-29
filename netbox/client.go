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

func (c Client) performGET(path string, query string) []byte {
	if len(query) == 0 {
		query = "?limit=2000"
	} else {
		query = query + "&limit=2000"
	}

	requestUrl := fmt.Sprintf("%v%v%v", c.conf.Netbox.URL, path, query)
	req, _ := http.NewRequest("GET", requestUrl, nil)
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

	body, _ := io.ReadAll(res.Body)
	bodyStr := string(body)
	if strings.TrimSpace(bodyStr) == "" {
		log.Fatal("Empty body returned")
	}
	log.Println(fmt.Sprintf("Response: %d - body: \n%s", res.StatusCode, bodyStr))

	return body
}

type ipamPrefixResponse struct {
	Count   int                `json:"count"`
	Results []model.IPAMPrefix `json:"results"`
}

func (c Client) GetIPAMPrefixes() []model.IPAMPrefix {
	response := ipamPrefixResponse{}
	err := json.Unmarshal(c.performGET("/ipam/prefixes/", ""), &response)
	if err != nil {
		panic(err)
	}

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

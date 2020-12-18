package ipl

import (
	"fmt"
	"io/ioutil"
	"peg.nu/nx/cache"
	"peg.nu/nx/config"
	"peg.nu/nx/model"
	"peg.nu/nx/tagparser"
	"peg.nu/nx/util"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type iplTarget struct {
	Enabled bool     `nx:"enable,ns:ipl"`
	Lists   []string `nx:"list,ns:ipl"`
}

type templateVars struct {
	Name        string
	IPs         []string
	GeneratedAt string
}

func GenerateIPLists(addresses []model.IPAddress, conf *config.NXConfig) {
	groupMap := make(map[string][]string)

	for _, address := range addresses {
		target := iplTarget{}
		tagparser.ParseTags(&target, address.Tags, address.Prefix.Tags)

		if !target.Enabled || len(target.Lists) == 0 {
			continue
		}

		slashIdx := strings.Index(address.Address, "/")
		strAddress := address.Address[:slashIdx]

		for _, list := range target.Lists {
			slice, ok := groupMap[list]

			if ok {
				groupMap[list] = append(slice, strAddress)
			} else {
				groupMap[list] = []string{strAddress}
			}
		}
	}

	now := time.Now()
	vars := templateVars{GeneratedAt: now.Format(time.RFC3339)}

	templateString, err := ioutil.ReadFile("templates/ip-list.tmpl")
	if err != nil {
		panic(err)
	}
	iplTemplate := template.Must(template.New("ipl").Parse(string(templateString)))

	cw := cache.New("generated/hashes/ipl.json")
	ignoreRegex := regexp.MustCompile("(?m)^# Generated at .*$")

	for group, ips := range groupMap {
		vars.Name = group
		vars.IPs = ips

		_, err := cw.WriteTemplate(
			fmt.Sprintf("generated/ipl/%s.ipl.txt", group),
			iplTemplate,
			vars,
			[]*regexp.Regexp{ignoreRegex},
			false,
		)
		if err != nil {
			panic(err)
		}
	}

	util.CleanDirectoryExcept("generated/ipl", cw.ProcessedFiles, conf)
	conf.UpdatedFiles = append(conf.UpdatedFiles, cw.UpdatedFiles...)
}

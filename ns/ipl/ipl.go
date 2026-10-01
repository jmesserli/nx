package ipl

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"peg.nu/nx/cache"
	"peg.nu/nx/config"
	"peg.nu/nx/model"
	"peg.nu/nx/util"
)

type templateVars struct {
	Name        string
	IPs         []string
	GeneratedAt string
}

func GenerateIPLists(addresses []model.IPAddress, conf *config.NXConfig) {
	groupMap := make(map[string][]string)

	for _, address := range addresses {
		if !address.Config.IPLEnabled || len(address.Config.IPLists) == 0 {
			continue
		}

		slashIdx := strings.Index(address.Address, "/")
		strAddress := address.Address[:slashIdx]

		for _, list := range address.Config.IPLists {
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

	templateString, err := os.ReadFile("templates/ip-list.tmpl")
	if err != nil {
		panic(err)
	}
	iplTemplate := template.Must(template.New("ipl").Parse(string(templateString)))
	ignoreRegexes := []*regexp.Regexp{
		regexp.MustCompile("(?m)^# Generated at .*$"),
	}

	cw := cache.New(iplTemplate, ignoreRegexes, false)

	for group, ips := range groupMap {
		vars.Name = group
		vars.IPs = ips

		_, err := cw.WriteTemplate(
			fmt.Sprintf("generated/ipl/%s.ipl.txt", group),
			vars,
		)
		if err != nil {
			panic(err)
		}
	}

	util.CleanDirectoryExcept("generated/ipl", cw.ProcessedFiles, conf)
	conf.UpdatedFiles = append(conf.UpdatedFiles, cw.UpdatedFiles...)
}

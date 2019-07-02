package util

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"

	"github.com/jmesserli/nx/config"
)

func MustConvertToBool(input string) bool {
	value, _ := strconv.ParseBool(input)

	return value
}

func ReverseSlice(a []string) []string {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}

	return a
}

func ExpandIPv6(ip net.IP) string {
	dst := make([]byte, hex.EncodedLen(len(ip)))
	_ = hex.Encode(dst, ip)
	return string(dst)
}

func CleanDirectory(directory string) {
	// clean zone directory
	dir, err := ioutil.ReadDir(directory)
	if err != nil {
		panic(err)
	}
	for _, d := range dir {
		os.RemoveAll(fmt.Sprintf("%s/%s", directory, d.Name()))
	}
}

func SliceContainsString(slice []string, value string) bool {
	for _, entry := range slice {
		if entry == value {
			return true
		}
	}

	return false
}

func FindMasterForZone(conf config.NXConfig, zone string) *config.MasterConfig {
	for _, master := range conf.Namespaces.DNS.Masters {
		if SliceContainsString(master.Zones, zone) {
			return &master
		}
	}

	return nil
}

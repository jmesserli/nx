package util

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"peg.nu/nx/config"
	"peg.nu/nx/model"
	"time"
)

var logger = log.New(os.Stdout, "[util] ", log.LstdFlags)

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

func CleanDirectoryExcept(directory string, exceptions []string, conf *config.NXConfig) {
	fileInfos, err := ioutil.ReadDir(directory)
	if err != nil {
		panic(err)
	}

	for _, fileInfo := range fileInfos {
		name := fmt.Sprintf("%s/%s", directory, fileInfo.Name())
		if SliceContainsString(exceptions, name) {
			continue
		}

		logger.Printf("Removing file %s\n", name)
		conf.UpdatedFiles = append(conf.UpdatedFiles, name)
		_ = os.RemoveAll(name)
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

func IpAddressesLessFn(addresses []model.IPAddress) func(i, j int) bool {
	return func(i, j int) bool {
		return CompareCIDRStrings(addresses[i].Address, addresses[j].Address)
	}
}

func CompareCIDRStrings(first, second string) bool {
	firstBytes := cidrStringToByteSlice(first)
	secondBytes := cidrStringToByteSlice(second)

	for i := 0; i < len(firstBytes); i++ {
		if firstBytes[i] < secondBytes[i] {
			return true
		}
		if firstBytes[i] > secondBytes[i] {
			return false
		}
	}

	return false
}

func cidrStringToByteSlice(cidrString string) []byte {
	ip, _, err := net.ParseCIDR(cidrString)
	if err != nil {
		log.Fatal(err)
	}

	return ip
}

func DurationSince(msg string, start time.Time) {
	logger.Printf("%v took %v.\n", msg, time.Since(start))
}

func StartTracking(msg string) (string, time.Time) {
	return msg, time.Now()
}

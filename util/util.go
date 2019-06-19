package util

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
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

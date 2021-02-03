package dns

import (
	"peg.nu/nx/model"
	"testing"
)

func TestDomainNormalizing(t *testing.T) {
	t.Run("min-min", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1", "", "peg.nu"), makeAddress("vm-ns-1", "", "peg.nu"), t)
	})
	t.Run("max-min", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1.peg.nu", "", "peg.nu"), makeAddress("vm-ns-1", "", "peg.nu"), t)
	})
	t.Run("max-max", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1.bue39.peg.nu", "", "bue39.peg.nu"), makeAddress("vm-ns-1.bue39", "", "peg.nu"), t)
	})
	t.Run("mid-max", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1.bue39", "", "bue39.peg.nu"), makeAddress("vm-ns-1.bue39.bue39", "", "peg.nu"), t)
	})
	t.Run("maxadd-min", func(t *testing.T) {
		testNameFixing(makeAddress("plex.rack.farm v4", "", "rack.farm"), makeAddress("plex", "", "rack.farm"), t)
	})
	t.Run("minadd-min", func(t *testing.T) {
		testNameFixing(makeAddress("plex and some text", "", "rack.farm"), makeAddress("plex", "", "rack.farm"), t)
	})
	t.Run("add-min", func(t *testing.T) {
		testNameFixing(makeAddress("just some text", "", "peg.nu"), makeAddress("just", "", "peg.nu"), t)
	})
	t.Run("max-min", func(t *testing.T) {
		testNameFixing(makeAddress("plex.plox.rack.farm", "", "rack.farm"), makeAddress("plex.plox", "", "rack.farm"), t)
	})
	t.Run("min-umin", func(t *testing.T) {
		testNameFixing(makeAddress("nas", "", "intra"), makeAddress("nas", "", "intra"), t)
	})
	t.Run("max-umin", func(t *testing.T) {
		testNameFixing(makeAddress("nas.intra", "", "intra"), makeAddress("nas", "", "intra"), t)
	})
	t.Run("minadd-umin", func(t *testing.T) {
		testNameFixing(makeAddress("nas und so", "", "intra"), makeAddress("nas", "", "intra"), t)
	})
}

func TestDnsFieldFallback(t *testing.T) {
	t.Run("DnsName only", func(t *testing.T) {
		testDnsFieldPreference(makeAddress("fabianflu.ch", "", ""), "fabianflu.ch", t)
	})
	t.Run("Both fields", func(t *testing.T) {
		testDnsFieldPreference(makeAddress("fabianflu.ch", "peg.nu", ""), "fabianflu.ch", t)
	})
	t.Run("Description only", func(t *testing.T) {
		testDnsFieldPreference(makeAddress("", "peg.nu", ""), "peg.nu", t)
	})
}

func makeAddress(dnsName, description, zone string) DNSIP {
	return DNSIP{
		IP: &model.IPAddress{
			DnsName:     dnsName,
			Description: description,
		},

		ForwardZoneName: zone,
	}
}

func testNameFixing(original, expect DNSIP, t *testing.T) {
	updated := original
	FixFlattenAddress(&updated)

	if updated.IP.DnsName != expect.IP.GetName() {
		t.Errorf("Expected Name to be <%s>; but was <%s>", expect.IP.GetName(), updated.IP.GetName())
	}

	if updated.ForwardZoneName != expect.ForwardZoneName {
		t.Errorf("Expected Zone to be <%s>; but was <%s>", expect.ForwardZoneName, updated.ForwardZoneName)
	}
}
func testDnsFieldPreference(a DNSIP, expect string, t *testing.T) {

	if a.IP.GetName() != expect {
		t.Errorf("Expected Hostname to be <%s>; but was <%s>", expect, a.IP.GetName())

	}
}

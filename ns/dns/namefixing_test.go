package dns

import (
	"testing"

	"github.com/jmesserli/nx/netbox"
)

func TestDomainNormalizing(t *testing.T) {
	t.Run("min-min", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1", "peg.nu"), makeAddress("vm-ns-1", "peg.nu"), t)
	})
	t.Run("max-min", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1.peg.nu", "peg.nu"), makeAddress("vm-ns-1", "peg.nu"), t)
	})
	t.Run("max-max", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1.bue39.peg.nu", "bue39.peg.nu"), makeAddress("vm-ns-1.bue39", "peg.nu"), t)
	})
	t.Run("mid-max", func(t *testing.T) {
		testNameFixing(makeAddress("vm-ns-1.bue39", "bue39.peg.nu"), makeAddress("vm-ns-1.bue39.bue39", "peg.nu"), t)
	})
	t.Run("maxadd-min", func(t *testing.T) {
		testNameFixing(makeAddress("plex.rack.farm v4", "rack.farm"), makeAddress("plex", "rack.farm"), t)
	})
	t.Run("minadd-min", func(t *testing.T) {
		testNameFixing(makeAddress("plex and some text", "rack.farm"), makeAddress("plex", "rack.farm"), t)
	})
	t.Run("add-min", func(t *testing.T) {
		testNameFixing(makeAddress("just some text", "peg.nu"), makeAddress("just", "peg.nu"), t)
	})
	t.Run("max-min", func(t *testing.T) {
		testNameFixing(makeAddress("plex.plox.rack.farm", "rack.farm"), makeAddress("plex.plox", "rack.farm"), t)
	})
	t.Run("min-umin", func(t *testing.T) {
		testNameFixing(makeAddress("nas", "intra"), makeAddress("nas", "intra"), t)
	})
	t.Run("max-umin", func(t *testing.T) {
		testNameFixing(makeAddress("nas.intra", "intra"), makeAddress("nas", "intra"), t)
	})
	t.Run("minadd-umin", func(t *testing.T) {
		testNameFixing(makeAddress("nas und so", "intra"), makeAddress("nas", "intra"), t)
	})
}

func makeAddress(name, zone string) netbox.IPAddress {
	return netbox.IPAddress{
		Name: name,
		GenOptions: netbox.GenerateOptions{
			ForwardZoneName: zone,
		},
	}
}

func testNameFixing(original, expect netbox.IPAddress, t *testing.T) {
	updated := original
	FixFlattenAddress(&updated)

	if updated.Name != expect.Name {
		t.Errorf("Expected Name to be <%s>; but was <%s>", expect.Name, updated.Name)
	}

	if updated.GenOptions.ForwardZoneName != expect.GenOptions.ForwardZoneName {
		t.Errorf("Expected Zone to be <%s>; but was <%s>", expect.GenOptions.ForwardZoneName, updated.GenOptions.ForwardZoneName)
	}
}

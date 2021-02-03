package tagparser

import (
	"peg.nu/nx/model"
	"testing"

	"github.com/go-test/deep"
)

type TestingStruct struct {
	StringValue   string   `nx:"string,ns:test"`
	IntValue      int      `nx:"int,ns:test"`
	StringSlice   []string `nx:"strsl,ns:test"`
	OtherNs       string   `nx:"sons,ns:test2"`
	IntSlice      []int    `nx:"intsl,ns:test"`
	Empty         string
	NotOverridden string `nx:"nov,ns:test"`
	Boolean       bool   `nx:"bol,ns:test"`
}

func TestTagParsing(t *testing.T) {
	expected := TestingStruct{
		StringValue:   "someString",
		IntValue:      42,
		StringSlice:   []string{"strings", "may", "slice"},
		OtherNs:       "someOtherString",
		IntSlice:      []int{1, 2, 3},
		NotOverridden: "",
		Boolean:       false,
	}
	actual := TestingStruct{}
	pTags := []model.Tag{
		{Name: "nx:test:string[overriddenValue]"}, {Name: "nx:test:int[invalidInt]"}, {Name: "nx:test:int[42]"}, {Name: "nx:test:nov[false]"},
	}
	tags := []model.Tag{
		{Name: "nx:test:string[someString]"}, {Name: "nx:text:string[shouldBeIgnored]"},
		{Name: "nx:test:strsl[strings]"}, {Name: "nx:test:strsl[may]"}, {Name: "nx:test:strsl[slice]"},
		{Name: "nx:test2:sons[someOtherString]"},
		{Name: "nx:test:intsl[1]"}, {Name: "nx:test:intsl[2]"}, {Name: "nx:test:intsl[3]"}, {Name: "nx:test:intsl[invalidIntSlice]"},
		{Name: "nx:test:nov[]"},
		{Name: "nx:test:bol[false]"},
	}

	ParseTags(&actual, tags, pTags)

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Error(diff)
	}
}

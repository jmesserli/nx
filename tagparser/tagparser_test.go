package tagparser

import (
	"testing"

	"github.com/go-test/deep"
)

type TestingStruct struct {
	StringValue string   `nx:"string,ns:test"`
	IntValue    int      `nx:"int,ns:test"`
	StringSlice []string `nx:"strsl,ns:test"`
	OtherNs     string   `nx:"sons,ns:test2"`
	IntSlice    []int    `nx:"intsl,ns:test"`
	Empty       string
}

func TestTagParsing(t *testing.T) {
	expected := TestingStruct{
		StringValue: "someString",
		IntValue:    42,
		StringSlice: []string{"strings", "may", "slice"},
		OtherNs:     "someOtherString",
		IntSlice:    []int{1, 2, 3},
	}
	actual := TestingStruct{}
	pTags := []string{"nx:test:string[overriddenValue]", "nx:test:int[invalidInt]", "nx:test:int[42]"}
	tags := []string{
		"nx:test:string[someString]", "nx:text:string[shouldBeIgnored]",
		"nx:test:strsl[strings]", "nx:test:strsl[may]", "nx:test:strsl[slice]",
		"nx:test2:sons[someOtherString]",
		"nx:test:intsl[1]", "nx:test:intsl[2]", "nx:test:intsl[3]", "nx:test:intsl[invalidIntSlice]",
	}

	ParseTags(&actual, tags, pTags)

	if diff := deep.Equal(expected, actual); diff != nil {
		t.Error(diff)
	}
}

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var testData = []struct {
	Pattern        string
	Topic          string
	Payload        []byte
	ExpectedTags   map[string]string
	ExpectedFields map[string]interface{}
}{
	// non matching patterns result in empty field and tags
	{"/foo/\\w+", "/hi/there", []byte("1.3"), map[string]string{}, map[string]interface{}{}},
	// no groups will result in field name to be last topic
	{"/foo/\\w+", "/foo/bar", []byte("1.3"), map[string]string{}, map[string]interface{}{"bar": []byte("1.3")}},
	{"/foo/bar/\\w+", "/foo/bar/muh", []byte("1.4"), map[string]string{}, map[string]interface{}{"muh": []byte("1.4")}},
	{"/foo/bar/\\w+", "/foo/bar/muh", []byte("-5"), map[string]string{}, map[string]interface{}{"muh": []byte("-5")}},
	// unnamed group will be used as field name
	{"/foo/bar/(\\w+)", "/foo/bar/muh", []byte("1.4"), map[string]string{}, map[string]interface{}{"muh": []byte("1.4")}},
	{"/foo/(bar)/\\w+", "/foo/bar/muh", []byte("1.4"), map[string]string{}, map[string]interface{}{"bar": []byte("1.4")}},
	{"/foo/(\\w+)/\\w+", "/foo/bazz/muh", []byte("1.4"), map[string]string{}, map[string]interface{}{"bazz": []byte("1.4")}},
	// multiple unnamed groups will be joined in one field name
	{"/foo/(\\w+)/(\\w+)", "/foo/bazz/muh", []byte("1.4"), map[string]string{}, map[string]interface{}{"bazz.muh": []byte("1.4")}},
	// named groups will be used as tag
	{"/foo/(?P<YOLO>\\w+)/\\w+", "/foo/bar/muh", []byte("1.4"), map[string]string{"YOLO": "bar"}, map[string]interface{}{"muh": []byte("1.4")}},
	{"/foo/(?P<YOLO>\\w+)/(?P<ALF>\\w+)/\\w+", "/foo/bar/2/muh", []byte("1.4"), map[string]string{"YOLO": "bar", "ALF": "2"}, map[string]interface{}{"muh": []byte("1.4")}},
	// also works in combination with field groups
	{"/foo/(?P<YOLO>\\w+)/(?P<ALF>\\w+)/(\\w+)", "/foo/bar/2/muh", []byte("1.4"), map[string]string{"YOLO": "bar", "ALF": "2"}, map[string]interface{}{"muh": []byte("1.4")}},
	{"/foo/(?P<YOLO>\\w+)/(?P<ALF>\\w+)/(\\w+)/\\w+", "/foo/bar/2/muh/goo", []byte("1.4"), map[string]string{"YOLO": "bar", "ALF": "2"}, map[string]interface{}{"muh": []byte("1.4")}},
	{"/foo/(?P<YOLO>\\w+)/(?P<ALF>\\w+)/(\\w+)/(\\w+)", "/foo/bar/2/muh/goo", []byte("1.4"), map[string]string{"YOLO": "bar", "ALF": "2"}, map[string]interface{}{"muh.goo": []byte("1.4")}},
	// boolean payload will be processed like numeric payload
	{"/foo/\\w+", "/foo/bar", []byte("false"), map[string]string{}, map[string]interface{}{"bar": []byte("false")}},
	{"/foo/(?P<YOLO>\\w+)/(?P<ALF>\\w+)/(\\w+)/(\\w+)", "/foo/bar/2/muh/goo", []byte("true"), map[string]string{"YOLO": "bar", "ALF": "2"}, map[string]interface{}{"muh.goo": []byte("true")}},
	// payload that is neither numeric nor boolean will be converted to a tag and an extra boolean occurred field will be added
	{"/foo/\\w+", "/foo/bar", []byte("wat"), map[string]string{"bar": "wat"}, map[string]interface{}{"occurred": true}},
	// the tag name will be unnamed groupd like field names before
	{"/(foo)/\\w+", "/foo/bar", []byte("wat"), map[string]string{"foo": "wat"}, map[string]interface{}{"occurred": true}},
	{"/(foo)/(\\w+)", "/foo/bar", []byte("wat"), map[string]string{"foo.bar": "wat"}, map[string]interface{}{"occurred": true}},
	{"/(foo)/(?P<lol>\\d)/(\\w+)", "/foo/1/bar", []byte("wat"), map[string]string{"foo.bar": "wat", "lol": "1"}, map[string]interface{}{"occurred": true}},
}

func TestGatherInfluxData(t *testing.T) {
	for _, td := range testData {

		tags, fields := gatherInfluxData(td.Pattern, td.Topic, []byte(td.Payload))

		assert.Equal(t, len(td.ExpectedTags), len(tags))

		for k, _ := range td.ExpectedTags {
			assert.Equal(t, td.ExpectedTags[k], tags[k])
		}

		assert.Equal(t, len(fields), len(td.ExpectedFields))

		for k, _ := range td.ExpectedFields {
			assert.Equal(t, td.ExpectedFields[k], fields[k])
		}
	}
}

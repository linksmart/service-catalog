package mqttmatch

import "testing"

func TestMatch(t *testing.T) {
	shouldNotMatch := map[string]string{
		"foo":         "bar",
		"foo/bar":     "foo/bar/baz",
		"foo/+":       "foo",
		"foo/#":       "fooo/abcd/bar/1234",
		"foo/bar/baz": "foo/bar",
		"foo/bar/+":   "foo/bar",
		"+/+":         "foo",
		"+":           "/foo",
	}
	for filter, topic := range shouldNotMatch {
		if r := Match(filter, topic); r == true {
			t.Errorf("%s and %s should not match", filter, topic)
		}
	}

	shouldMatch := map[string]string{
		"foo":         "foo",
		"foo/bar":     "foo/bar",
		"foo/+":       "foo/bar",
		"foo2/+":      "foo2/",
		"foo/bar/+":   "foo/bar/baz",
		"foo/+/bar/+": "foo/abcd/bar/1234",
		"foo/#":       "foo/abcd/bar/1234",
		"foo2/#":      "foo2/abcd",
		"foo/+/bar/#": "foo/abcd/bar/1234/fooagain",
		"+/+":         "foo/bar",
		"+/#":         "foo/bar/baz",
		"#":           "foo/bar/baz",
		"/+":          "/foo",
		"+/+/+":       "/foo/bar",
	}
	for filter, topic := range shouldMatch {
		if r := Match(filter, topic); r == false {
			t.Errorf("%s and %s should match", filter, topic)
		}
	}
}

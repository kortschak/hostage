// Copyright ©2021 Dan Kortschak. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"reflect"
	"strings"
	"testing"
)

var testCases = []struct {
	name          string
	seq           []sequence
	wantMapping   map[string]interface{}
	wantFormatted string
}{
	{
		name:        "empty",
		seq:         nil,
		wantMapping: map[string]interface{}{},
		wantFormatted: `{
}
`,
	},
	{
		name: "one by one",
		seq: []sequence{
			{path: []string{"1"}, val: "text"},
		},
		wantMapping: map[string]interface{}{
			"1": "text",
		},
		wantFormatted: `{
	"1" = ("insertText:", "text");
}
`,
	},
	{
		name: "one by two",
		seq: []sequence{
			{path: []string{"1", "2"}, val: "text"},
		},
		wantMapping: map[string]interface{}{
			"1": map[string]interface{}{
				"2": "text",
			},
		},
		wantFormatted: `{
	"1" = {
		"2" = ("insertText:", "text");
	};
}
`,
	},
	{
		name: "two by one",
		seq: []sequence{
			{path: []string{"1"}, val: "text1"},
			{path: []string{"2"}, val: "text2"},
		},
		wantMapping: map[string]interface{}{
			"1": "text1",
			"2": "text2",
		},
		wantFormatted: `{
	"1" = ("insertText:", "text1");
	"2" = ("insertText:", "text2");
}
`,
	},
	{
		name: "two by two",
		seq: []sequence{
			{path: []string{"1", "2"}, val: "text1"},
			{path: []string{"1", "3"}, val: "text2"},
		},
		wantMapping: map[string]interface{}{
			"1": map[string]interface{}{
				"2": "text1",
				"3": "text2",
			},
		},
		wantFormatted: `{
	"1" = {
		"2" = ("insertText:", "text1");
		"3" = ("insertText:", "text2");
	};
}
`,
	},
}

type sequence struct {
	path []string
	val  string
}

func Test(t *testing.T) {
	for _, test := range testCases {
		gotMapping := make(map[string]interface{})
		for _, s := range test.seq {
			insert(gotMapping, s.val, s.path...)
		}
		if !reflect.DeepEqual(gotMapping, test.wantMapping) {
			t.Errorf("unexpected result for %q:\ngot: %#v\nwant:%#v",
				test.name, gotMapping, test.wantMapping)
		}
		var buf strings.Builder
		format(&buf, gotMapping, 0)
		gotFormatted := buf.String()
		if gotFormatted != test.wantFormatted {
			t.Errorf("unexpected result for %q:\ngot:\n%s\nwant:\n%s",
				test.name, gotFormatted, test.wantFormatted)
		}
	}
}

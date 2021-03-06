package main

import (
	"testing"

	"github.com/neoul/yangtree"
)

func Test_RPath2XPath(t *testing.T) {
	rc := loadSchema(*yangfiles, *dir, *excludes)

	rpath := []string{
		"/modules-state/module=yangtree,2020-08-18/namespace",
		"/modules-state/module=yangtree,2020-08-18/",
		"/modules-state/module=yangtree,2020-08-18",
		"/modules-state/module",
		"/modules-state/module=1/1,2020-08-18/",
		"/modules-state/module=1/1/1,2020-08-18/",
		"/modules-state/module=A,2020-08-18/UNKNOWN",
		"/modules-state/UNKNOWN",
	}
	xpath := []string{
		"modules-state/module[name=yangtree][revision=2020-08-18]/namespace",
		"modules-state/module[name=yangtree][revision=2020-08-18]",
		"modules-state/module[name=yangtree][revision=2020-08-18]",
		"modules-state/module",
		"modules-state/module[name=1/1][revision=2020-08-18]",
		"modules-state/module[name=1/1/1][revision=2020-08-18]",
		"modules-state/module[name=A][revision=2020-08-18/UNKNOWN]", // unmatched UNKNOWN becomes a key
		"",
	}
	wanterr := []bool{
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		true,
	}

	type test struct {
		schema  *yangtree.SchemaNode
		rpath   *string
		want    string
		wantErr bool
	}
	var tests []test
	for i := range rpath {
		tests = append(tests, test{
			schema:  rc.schemaData,
			rpath:   &rpath[i],
			want:    xpath[i],
			wantErr: wanterr[i],
		})
	}
	for _, tt := range tests {
		t.Run(" "+*tt.rpath, func(t *testing.T) {
			s, got, err := RPath2XPath(tt.schema, tt.rpath)
			if (err != nil) != tt.wantErr {
				t.Errorf("RPath2XPath() error = %v, wantErr %v, schema=%v", err, tt.wantErr, s)
				return
			}
			if got != tt.want {
				t.Errorf("RPath2XPath() = %v, want %v, schema=%v", got, tt.want, s)
			}
		})
	}
}

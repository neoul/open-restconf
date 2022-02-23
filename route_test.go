package main

import (
	"testing"

	"github.com/neoul/yangtree"
)

func Test_RPath2XPath(t *testing.T) {
	rc := loadRESTCONFSchema(*yangfiles, *dir, *excludes)

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
		"/modules-state/module=A,2020-08-18/UNKNOWN",
		"/modules-state/module=A,2020-08-18/UNKNOWN",
		"/modules-state/UNKNOWN",
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
			schema: rc.schemaData,
			rpath:  &rpath[i],
			want:   xpath[i],
		})
	}
	for _, tt := range tests {
		t.Run(*tt.rpath, func(t *testing.T) {
			got, err := RPath2XPath(tt.schema, tt.rpath)
			if (err != nil) != tt.wantErr {
				t.Errorf("RPath2XPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RPath2XPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

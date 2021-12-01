package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gofiber/fiber"
	"github.com/neoul/yangtree"
	"github.com/spf13/pflag"
)

type RestconfCtrl struct {
	sync.RWMutex
	yangtree.DataNode

	status        int // HTTP response status
	errors        []yangtree.DataNode
	curnode       []yangtree.DataNode
	isGroupSearch bool
}

var (
	errorSchema, restconfSchema *yangtree.SchemaNode
)

var (
	bindAddr      = pflag.StringP("bind-address", "b", ":8080", "bind to address:port")
	startupFile   = pflag.String("startup", "", "startup data formatted to ietf-json or yaml")
	startupFormat = pflag.String("startup-format", "json", "startup data format [xml, json, yaml]")
	help          = pflag.BoolP("help", "h", false, "help for gnmid")
	yangfiles     = pflag.StringArrayP("files", "f", []string{}, "yang files to load")
	dir           = pflag.StringArrayP("dir", "d", []string{}, "directories to search yang includes and imports")
	excludes      = pflag.StringArrayP("exclude", "e", []string{}, "yang modules to be excluded from path generation")
)

func main() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
	if *help {
		fmt.Fprintf(pflag.CommandLine.Output(), "\n open-restconf server\n")
		fmt.Fprintf(pflag.CommandLine.Output(), "\n")
		fmt.Fprintf(pflag.CommandLine.Output(), " usage: %s [flag]\n", os.Args[0])
		fmt.Fprintf(pflag.CommandLine.Output(), "\n")
		pflag.PrintDefaults()
		fmt.Fprintf(pflag.CommandLine.Output(), "\n")
		return
	}
	file := []string{
		"modules/ietf-yang-library@2016-06-21.yang",
		"modules/ietf-restconf@2017-01-26.yang",
		// "modules/ietf-interfaces@2018-02-20.yang",
		// "modules/iana-if-type@2017-01-19.yang",

		// "modules/example/example-jukebox.yang",
		// "modules/example/example-mod.yang",
		// "modules/example/example-ops.yang",
		// "modules/example/example-actions.yang",
	}
	file = append(file, *yangfiles...)
	rootSchema, err := yangtree.Load(file, *dir, *excludes, yangtree.YANGTreeOption{YANGLibrary2016: true})
	if err != nil {
		if merr, ok := err.(yangtree.MultipleError); ok {
			for i := range merr {
				log.Fatalf("restconf: error[%d] in loading: %v", i, merr[i])
			}
		} else {
			log.Fatalf("restconf: error in loading: %v", err)
		}
	}
	// loading restconf.error
	yangerrorSchema := rootSchema.ExtSchema["yang-errors"]
	if yangerrorSchema == nil {
		log.Fatalf("restconf: unable to load restconf schema")
	}

	// loading restconf.top
	yangapiSchema := rootSchema.ExtSchema["yang-api"]
	if yangapiSchema == nil {
		log.Fatalf("restconf: unable to load restconf schema")
	}
	var ylibrev string
	if rootSchema.GetYangLibrary().Exist("module[name=ietf-yang-library][revision=2016-06-21]") {
		ylibrev = "2016-06-21"
	}
	errorSchema = yangerrorSchema.GetSchema("errors")
	restconfSchema = yangapiSchema.GetSchema("restconf")

	// All schema nodes in root schema move to /restconf/data or /restconf/operations nodes.
	for i := range rootSchema.Children {
		if rootSchema.Children[i].RPC != nil {
			restconfSchema.GetSchema("operations").Append(true, rootSchema.Children[i])
		} else {
			restconfSchema.GetSchema("data").Append(true, rootSchema.Children[i])
		}
	}
	restroot, err := yangtree.NewWithValue(restconfSchema,
		map[interface{}]interface{}{
			"data":                 map[interface{}]interface{}{},
			"operations":           nil,
			"yang-library-version": ylibrev,
		})
	if err != nil {
		log.Fatalf("restconf: %v", err)
	}
	dataroot := restroot.Get("data")
	if *startupFile != "" {
		var file *os.File
		file, err = os.Open(*startupFile)
		if err != nil {
			log.Fatalf("restconf: %v", err)
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("restconf: %v", err)
		}
		file.Close()
		switch *startupFormat {
		case "yaml":
			if err := yangtree.UnmarshalYAML(dataroot, b); err != nil {
				log.Fatalf("restconf: %v", err)
			}
		case "xml":
			if err := yangtree.UnmarshalXML(dataroot, b); err != nil {
				log.Fatalf("restconf: %v", err)
			}
		case "json":
			if err := yangtree.UnmarshalJSON(dataroot, b); err != nil {
				log.Fatalf("restconf: %v", err)
			}
		}
	}
	// if j, _ := yangtree.MarshalYAML(dataroot); len(j) > 0 {
	// 	fmt.Println(string(j))
	// }

	app := fiber.New()

	rctrl := &RestconfCtrl{
		DataNode:      restroot,
		curnode:       nil,
		isGroupSearch: false,
	}
	if err := InstallRouteRoot(app, rctrl); err != nil {
		log.Fatalf("restconf: %v", err)
	}

	// register restconf host-meta info.
	if err := InstallRouteHostMeta(app, rctrl); err != nil {
		log.Fatalf("restconf: %v", err)
	}

	log.Println("[modules loaded]")
	mnames := make([]string, 0, len(rootSchema.Modules.Modules))
	for k := range rootSchema.Modules.Modules {
		if strings.Contains(k, "@") {
			mnames = insertionSort(mnames, k)
		}
	}
	for i := range mnames {
		log.Println(" -", mnames[i])
	}
	log.Println("[submodules loaded]")
	mnames = mnames[:0]
	for k := range rootSchema.Modules.SubModules {
		if strings.Contains(k, "@") {
			mnames = insertionSort(mnames, k)
		}
	}
	for i := range mnames {
		log.Println(" -", mnames[i])
	}
	log.Println("")

	// start fiber.app
	if err := app.Listen(*bindAddr); err != nil {
		log.Fatalf("restconf: %v", err)
	}
}

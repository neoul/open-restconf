// RFC8040 RESTCONF Protocol implementation
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
	"github.com/gofiber/fiber/middleware/logger"
	"github.com/gofiber/fiber/middleware/requestid"
	"github.com/neoul/yangtree"
	"github.com/spf13/pflag"
)

type RESTCtrl struct {
	sync.RWMutex
	DataRoot         yangtree.DataNode // /restconf/data
	schemaError      *yangtree.SchemaNode
	schemaErrors     *yangtree.SchemaNode
	schemaRESTCONF   *yangtree.SchemaNode
	schemaData       *yangtree.SchemaNode
	schemaOperations *yangtree.SchemaNode
	rootSchema       *yangtree.SchemaNode
	yangLibVersion   string
}

var (
	bindAddr      = pflag.StringP("bind-address", "b", ":8080", "bind to address:port")
	startupFile   = pflag.String("startup", "", "startup data formatted to ietf-json or yaml")
	startupFormat = pflag.String("startup-format", "json", "startup data format [xml, json, yaml]")
	help          = pflag.BoolP("help", "h", false, "help for gnmid")
	yangfiles     = pflag.StringArrayP("files", "f", []string{}, "yang files to load")
	dir           = pflag.StringArrayP("dir", "d", []string{}, "directories to search yang includes and imports")
	excludes      = pflag.StringArrayP("exclude", "e", []string{}, "yang modules to be excluded from path generation")

	restfiles = []string{
		"modules/ietf-yang-library@2016-06-21.yang",
		"modules/ietf-restconf@2017-01-26.yang",
		// "modules/ietf-interfaces@2018-02-20.yang",
		// "modules/iana-if-type@2017-01-19.yang",

		// "modules/example/example-jukebox.yang",
		// "modules/example/example-mod.yang",
		// "modules/example/example-ops.yang",
		// "modules/example/example-actions.yang",
	}
)

func loadSchema(file, dir, excludes []string) *RESTCtrl {
	var err error
	rc := &RESTCtrl{}
	file = append(file, restfiles...)
	rc.rootSchema, err = yangtree.Load(file, dir, excludes, yangtree.YANGTreeOption{YANGLibrary2016: true})
	if err != nil {
		if merr, ok := err.(yangtree.MultipleError); ok {
			for i := range merr {
				log.Fatalf("restconf: error[%d] in loading: %v", i, merr[i])
			}
		} else {
			log.Fatalf("restconf: error in loading: %v", err)
		}
	}
	// load restconf.errors.
	yangerrorSchema := rc.rootSchema.ExtSchema["yang-errors"]
	if yangerrorSchema == nil {
		log.Fatalf("restconf: unable to load yang-errors schema")
	}
	rc.schemaErrors = yangerrorSchema.GetSchema("errors")
	if rc.schemaErrors == nil {
		log.Fatalf("restconf: unable to load yang-errors/errors schema")
	}
	rc.schemaError = rc.schemaErrors.GetSchema("error")
	if rc.schemaError == nil {
		log.Fatalf("restconf: unable to load yang-errors/errors/error schema")
	}

	// load restconf.top.
	yangapiSchema := rc.rootSchema.ExtSchema["yang-api"]
	if yangapiSchema == nil {
		log.Fatalf("restconf: unable to load yang-api schema")
	}
	if rc.rootSchema.GetYangLibrary().Exist("module[name=ietf-yang-library][revision=2016-06-21]") {
		rc.yangLibVersion = "2016-06-21"
	}

	// move all schema nodes in the root schema to /restconf/data or /restconf/operations nodes.
	rc.schemaRESTCONF = yangapiSchema.GetSchema("restconf")
	if rc.schemaRESTCONF == nil {
		log.Fatalf("restconf: unable to load restconf schema")
	}
	rc.schemaOperations = rc.schemaRESTCONF.GetSchema("operations")
	if rc.schemaOperations == nil {
		log.Fatalf("restconf: unable to load restconf/data schema")
	}
	rc.schemaData = rc.schemaRESTCONF.GetSchema("data")
	if rc.schemaData == nil {
		log.Fatalf("restconf: unable to load restconf/data schema")
	}
	for i := range rc.rootSchema.Children {
		if rc.rootSchema.Children[i].RPC != nil {
			rc.schemaOperations.Append(true, rc.rootSchema.Children[i])
		} else {
			rc.schemaData.Append(true, rc.rootSchema.Children[i])
		}
	}

	return rc
}

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
	rc := loadSchema(*yangfiles, *dir, *excludes)

	// create the data node.
	dataroot, err := yangtree.New(rc.schemaData)
	if err != nil {
		log.Fatalf("restconf: unable to create the restconf data root: %v", err)
	}

	// load yanglibrary
	library := rc.rootSchema.GetYangLibrary()
	if _, err := dataroot.Insert(library, nil); err != nil {
		log.Fatalf("restconf: unable to add the yanglibrary: %v", err)
	}

	// load startup data.
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

	// create the restconf service
	app := fiber.New(fiber.Config{
		ErrorHandler: errhandler,
	})
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(requestid.New()) // add requestid
	rc.DataRoot = dataroot
	// register restconf host-meta info.
	if err := InstallRouteHostMeta(app, rc); err != nil {
		log.Fatalf("restconf: %v", err)
	}
	if err := InstallRouteRESTCONF(app, rc); err != nil {
		log.Fatalf("restconf: %v", err)
	}
	if err := InstallRouteSchemaPath(app, rc); err != nil {
		log.Fatalf("restconf: %v", err)
	}
	if err := InstallRouteYANGModules(app, library, *dir); err != nil {
		log.Fatalf("restconf: %v", err)
	}
	if err := InstallRouteYANGModules(app, library, *yangfiles); err != nil {
		log.Fatalf("restconf: %v", err)
	}
	if err := InstallRouteYANGModules(app, library, restfiles); err != nil {
		log.Fatalf("restconf: %v", err)
	}

	log.Println("[modules loaded]")
	mnames := make([]string, 0, len(rc.rootSchema.Modules.Modules))
	for k := range rc.rootSchema.Modules.Modules {
		if strings.Contains(k, "@") {
			mnames = InsertionSort(mnames, k)
		}
	}
	for i := range mnames {
		log.Println(" -", mnames[i])
	}
	log.Println("[submodules loaded]")
	mnames = mnames[:0]
	for k := range rc.rootSchema.Modules.SubModules {
		if strings.Contains(k, "@") {
			mnames = InsertionSort(mnames, k)
		}
	}
	for i := range mnames {
		log.Println(" -", mnames[i])
	}
	log.Println("")

	// nodes, _ := yangtree.Find(library, "module")
	// node, _ := yangtree.ConvertToGroup(nodes[0].Schema(), nodes)
	// b, _ := yangtree.MarshalYAMLIndent(node, "", " ") // yangtree.RepresentItself{}
	// fmt.Println(string(b))

	// start fiber.app
	if err := app.Listen(*bindAddr); err != nil {
		log.Fatalf("restconf: %v", err)
	}
}

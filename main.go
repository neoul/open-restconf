package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/gofiber/fiber"
	"github.com/neoul/yangtree"
	"github.com/spf13/pflag"
)

// GenerateRoutePath() generates a fiber route path and an yang data path format from a schema node.
func GenerateRoutePath(schema *yangtree.SchemaNode, prefixTagging bool) (routePath, searchPath []string) {
	var routeElem strings.Builder
	var searchElem strings.Builder
	if prefixTagging && schema.Prefix != nil {
		routeElem.WriteString(schema.Prefix.Name)
		routeElem.WriteString("\\:")
		routeElem.WriteString(schema.Name)

		searchElem.WriteString(schema.Prefix.Name)
		searchElem.WriteString(":")
		searchElem.WriteString(schema.Name)
	} else {
		routeElem.WriteString(schema.Name)

		searchElem.WriteString(schema.Name)
	}
	routePath = append(routePath, "/"+routeElem.String())
	searchPath = append(searchPath, searchElem.String())

	if len(schema.Keyname) == 0 {
		return
	} else {
		comma := false
		routeElem.WriteString("=")
		for i := range schema.Keyname {
			if comma {
				routeElem.WriteString(",")
			}
			comma = true
			routeElem.WriteString(":")
			routeElem.WriteString(schema.Name)
			routeElem.WriteString("\\:")
			routeElem.WriteString(schema.Keyname[i])

			searchElem.WriteString("[")
			searchElem.WriteString(schema.Keyname[i])
			searchElem.WriteString("=%s]")
		}
	}
	routePath = append(routePath, "/"+routeElem.String())
	searchPath = append(searchPath, searchElem.String())
	return
}

type DataStore struct {
	sync.RWMutex
	yangtree.DataNode

	searchnode    []yangtree.DataNode
	isGroupSearch bool
}

var (
	errorSchema, restconfSchema *yangtree.SchemaNode
)

func SetRoute(router fiber.Router, schema *yangtree.SchemaNode, datastore *DataStore) error {
	routePath, searchPath := GenerateRoutePath(schema, false)
	log.Println(schema.Path())
	for i := range routePath {
		if schema.IsRPC() {
			router.Post(routePath[i], func(c *fiber.Ctx) error {
				rpc := schema.RPC
				if rpc.Input != nil {
					var b interface{}
					if err := c.BodyParser(&b); err != nil {
						return err
					}
				}
				if rpc.Output != nil {

				}
				return nil
			})
		} else if schema.IsDir() {
			ngroup := router.Group(routePath[i], func(c *fiber.Ctx) error {
				pname := c.Route().Params
				p := ""
				if len(pname) > 0 {
					pdata := make([]interface{}, len(pname))
					for j := range pname {
						pdata[j] = c.Params(pname[j])
					}
					p = fmt.Sprintf(searchPath[i], pdata...)
					if schema.IsList() {
						datastore.isGroupSearch = false
					}
				} else {
					p = searchPath[0]
					if schema.IsList() {
						datastore.isGroupSearch = true
					}
				}

				var rnodes []yangtree.DataNode
				for j := range datastore.searchnode {
					if schema.Name == datastore.searchnode[j].Name() {
						// select matched nodes with the params.
						if p == datastore.searchnode[j].ID() {
							rnodes = append(rnodes, datastore.searchnode[j])
						}
					} else {
						n, err := yangtree.Find(datastore.searchnode[j], p)
						if err != nil {
							return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
								"error":   true,
								"message": err.Error(),
							})
						}
						rnodes = append(rnodes, n...)
					}
				}
				datastore.searchnode = rnodes
				if len(datastore.searchnode) == 0 {
					return ErrorResponse(c, ETypeApplication, ETagDataMissing, nil)
				}
				log.Println("=> ", c.Path(), p, c.Route().Params, "RESULT", datastore.searchnode)
				err := c.Next()

				return err
			})
			for j := range schema.Children {
				if err := SetRoute(ngroup, schema.Children[j], datastore); err != nil {
					return err
				}
			}
			router.Get(routePath[i], func(c *fiber.Ctx) error { return nil })
		} else {
			router.Get(routePath[i], func(c *fiber.Ctx) error {
				log.Println(schema.Name, c.Response().StatusCode())
				if len(datastore.searchnode) == 0 {
					return nil
				}
				pname := c.Route().Params
				p := ""
				if len(pname) > 0 {
					pdata := make([]interface{}, len(pname))
					for j := range pname {
						pdata[j] = c.Params(pname[j])
					}
					p = fmt.Sprintf(searchPath[i], pdata...)
				} else {
					p = searchPath[i]
					if schema.IsListable() {
						datastore.isGroupSearch = true
					}
				}
				var rnodes []yangtree.DataNode
				for j := range datastore.searchnode {
					n, err := yangtree.Find(datastore.searchnode[j], p)
					if err != nil {
						return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
							"error":   true,
							"message": err.Error(),
						})
					}
					rnodes = append(rnodes, n...)
				}
				datastore.searchnode = rnodes
				return nil
			})
		}
	}
	return nil
}

var (
	bindAddr      = pflag.StringP("bind-address", "b", ":8080", "bind to address:port")
	startupFile   = pflag.String("startup", "", "startup data formatted to ietf-json or yaml")
	startupFormat = pflag.String("startup-format", "json", "startup data format [xml, json, yaml]")
	help          = pflag.BoolP("help", "h", false, "help for gnmid")
	yangfiles     = pflag.StringArrayP("files", "f", []string{}, "yang files to load")
	dir           = pflag.StringArrayP("dir", "d", []string{}, "directories to search yang includes and imports")
	excludes      = pflag.StringArrayP("exclude", "e", []string{}, "yang modules to be excluded from path generation")
)

func insertionSort(ss []string, s string) []string {
	i := sort.SearchStrings(ss, s)
	ss = append(ss, "")
	copy(ss[i+1:], ss[i:])
	ss[i] = s
	return ss
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

	datastore := &DataStore{
		DataNode:      restroot,
		searchnode:    nil,
		isGroupSearch: false,
	}
	top := app.Group("/restconf", func(c *fiber.Ctx) error {
		log.Println(c.Path())
		datastore.Lock()
		defer datastore.Unlock()
		datastore.isGroupSearch = false
		datastore.searchnode = []yangtree.DataNode{restroot}
		datastore.DataNode = restroot
		if err := c.Next(); err != nil {
			log.Printf("restconf: %s %s: %v\n", c.Method(), c.Path(), err)
			return ErrorResponse(c, ETypeProtocol, ETagBadElement, err)
		} else {
			switch c.Response().StatusCode() {
			case fiber.StatusConflict, fiber.StatusNotFound:
				return ErrorResponse(c, ETypeApplication, ETagDataMissing,
					fiber.NewError(fiber.StatusNotFound, "resouce not found"))
			}
		}
		c.Set("Server", "open-restconf")
		c.Set("Cache-Control", "no-cache")

		switch c.Method() {
		case "GET":
			if len(datastore.searchnode) == 0 {
				return ErrorResponse(c, ETypeApplication, ETagDataMissing, nil)
			}
			var rnode yangtree.DataNode
			if datastore.isGroupSearch {
				rnode, err = yangtree.ConvertToGroup(datastore.searchnode[0].Schema(), datastore.searchnode)
				if err != nil {
					return ErrorResponse(c, ETypeApplication, ETagOperationFailed, err)
				}
			} else {
				rnode = datastore.searchnode[0]
			}
			return GetResponse(c, rnode)
		case "POST":
			return nil
		default:
			return nil
		}
	})
	app.Get("/restconf", func(c *fiber.Ctx) error {
		return nil
	})

	for i := range restconfSchema.Children {
		if err := SetRoute(top, restconfSchema.Children[i], datastore); err != nil {
			log.Fatalf("restconf: %v", err)
		}
	}

	// register restconf host-meta info.
	app.Get("/.well-known/host-meta", GetHostMeta)

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

func GetHostMeta(c *fiber.Ctx) error {
	log.Println(c.BaseURL(), c.Path())
	// FIXME - add a link for the restconf access point
	c.Links("http://localhost:300/restconf")
	hdr := &(c.Response().Header)
	hdr.Add("Content-Type", "application/xrd+xml")
	hostmeta :=
		`<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
<Link rel='restconf' href='%s'/>
</XRD>`
	fmt.Fprintf(c, hostmeta, "/restconf")
	return nil
}

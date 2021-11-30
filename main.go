package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gofiber/fiber"
	"github.com/neoul/yangtree"
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

func SetRoute(router fiber.Router, schema *yangtree.SchemaNode, datastore *DataStore) error {
	routePath, searchPath := GenerateRoutePath(schema, false)
	log.Println(schema.Path(), "====> ", routePath)
	for i := range routePath {
		if schema.IsDir() {
			ngroup := router.Group(routePath[i], func(c *fiber.Ctx) error {
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
				log.Println("=> ", c.Path(), p, c.Route().Params, "RESULT", datastore.searchnode)
				return c.Next()
			})
			for j := range schema.Children {
				if err := SetRoute(ngroup, schema.Children[j], datastore); err != nil {
					return err
				}
			}
		} else {
			router.Get(routePath[i], func(c *fiber.Ctx) error {
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

func main() {
	yangfiles := []string{
		"modules/ietf-yang-library@2016-06-21.yang",
		"modules/ietf-restconf@2017-01-26.yang",
		// "modules/ietf-interfaces@2018-02-20.yang",
		// "modules/iana-if-type@2017-01-19.yang",

		"modules/example/example-jukebox.yang",
		// "modules/example/example-mod.yang",
		// "modules/example/example-ops.yang",
		// "modules/example/example-actions.yang",
	}
	dir := []string{"modules"}
	excluded := []string{}
	rootSchema, err := yangtree.Load(yangfiles, dir, excluded, yangtree.YANGTreeOption{YANGLibrary2016: true})
	if err != nil {
		if merr, ok := err.(yangtree.MultipleError); ok {
			for i := range merr {
				log.Fatalf("restconf: error[%d] in loading: %v", i, merr[i])
			}
		} else {
			log.Fatalf("restconf: error in loading: %v", err)
		}
	}
	// loading restconf.top
	yangapiSchema := rootSchema.ExtSchema["yang-api"]
	var ylibrev string
	if rootSchema.GetYangLibrary().Exist("module[name=ietf-yang-library][revision=2016-06-21]") {
		ylibrev = "2016-06-21"
	}
	restconfSchema := yangapiSchema.GetSchema("restconf")
	restconfSchema.GetSchema("data").Append(true, rootSchema.Children...)
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
	{
		var file *os.File
		file, err = os.Open("testdata/jukebox.yaml")
		if err != nil {
			log.Fatalf("restconf: %v", err)
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("restconf: %v", err)
		}
		file.Close()
		if err := yangtree.UnmarshalYAML(dataroot, b); err != nil {
			log.Fatalf("restconf: %v", err)
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
		datastore.Lock()
		defer datastore.Unlock()
		datastore.isGroupSearch = false
		datastore.searchnode = []yangtree.DataNode{restroot}
		datastore.DataNode = restroot
		if err := c.Next(); err != nil {
			log.Fatalf("restconf: %v", err)
		}
		c.Set("Server", "open-restconf")
		c.Set("Cache-Control", "no-cache")

		var marshal func(node yangtree.DataNode, option ...yangtree.Option) ([]byte, error)
		marshal = yangtree.MarshalXML
		accepts := c.Accepts("text/json", "text/yaml", "text/xml",
			"application/xml", "application/json", "application/yaml",
			"application/yang-data+xml", "application/yang-data+json", "application/yang-data+yaml")
		switch {
		case strings.HasSuffix(accepts, "xml"):
			c.Set("Content-Type", accepts)
			marshal = yangtree.MarshalXML
		case strings.HasSuffix(accepts, "json"):
			c.Set("Content-Type", accepts)
			marshal = yangtree.MarshalJSON
		case strings.HasSuffix(accepts, "yaml"):
			c.Set("Content-Type", accepts)
			marshal = yangtree.MarshalYAML
		default:
			return c.Status(fiber.StatusNotAcceptable).JSON(fiber.Map{
				"error":   true,
				"message": fmt.Errorf("acceptable contents: json, yaml, xml"),
			})
		}
		if len(datastore.searchnode) == 0 {
			return nil
		}
		var rnode yangtree.DataNode
		if datastore.isGroupSearch {
			rnode, err = yangtree.ConvertToGroup(datastore.searchnode[0].Schema(), datastore.searchnode)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": true,
					"msg":   err.Error(),
				})
			}
		} else {
			rnode = datastore.searchnode[0]
		}
		b, err := marshal(rnode)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": true,
				"msg":   err.Error(),
			})
		}
		return c.Send(b)
	})
	for i := range restconfSchema.Children {
		if err := SetRoute(top, restconfSchema.Children[i], datastore); err != nil {
			log.Fatalf("restconf: %v", err)
		}
	}

	// register restconf host-meta info.
	app.Get("/.well-known/host-meta", HandleHostMeta)

	// start fiber.app
	app.Listen(":3000")
}

func HandleHostMeta(c *fiber.Ctx) error {
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

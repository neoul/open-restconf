package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber"
	"github.com/neoul/yangtree"
)

func rpathKey2xpathKey(xpathB *strings.Builder, schema *yangtree.SchemaNode, keystr *string) {
	var keynodes []string
	if len(schema.Keyname) == 1 {
		keynodes = []string{*keystr}
	} else {
		keynodes = strings.Split(*keystr, ",")
	}
	for j := range keynodes {
		if j < len(schema.Keyname) {
			xpathB.WriteString("[")
			xpathB.WriteString(schema.Keyname[j])
			xpathB.WriteString("=")
			xpathB.WriteString(keynodes[j])
			xpathB.WriteString("]")
		}
	}
}

// RPath2XPath() converts RESTCONF URI(Route Path) to XPath
func RPath2XPath(schema *yangtree.SchemaNode, uri *string) (string, error) {
	var xpathB strings.Builder
	pathnodes := strings.Split(*uri, "/")
	snode := schema
	var keystr string
	for i := range pathnodes {
		if index := strings.Index(pathnodes[i], "="); index >= 0 {
			if keystr != "" {
				return "", fmt.Errorf("failed to extract the key for %s", snode)
			}
			s := snode.GetSchema(pathnodes[i][:index])
			if s == nil {
				return "", fmt.Errorf("unable to find schema %s", pathnodes[i][:index])
			}
			snode = s
			keystr = pathnodes[i][index+1:]
			xpathB.WriteString(s.Name)
		} else if pathnodes[i] == "" {
			continue
		} else {
			s := snode.GetSchema(pathnodes[i])
			if s == nil {
				if len(keystr) > 0 {
					keystr = keystr + "/" + pathnodes[i]
					continue
				}
				return "", fmt.Errorf("unable to find schema %s", pathnodes[i])
			}
			if keystr != "" {
				rpathKey2xpathKey(&xpathB, snode, &keystr)
				xpathB.WriteString("/")
				keystr = ""
			}
			snode = s
			xpathB.WriteString(pathnodes[i])
			if len(pathnodes) != i+1 {
				xpathB.WriteString("/")
			}
		}
	}
	if keystr != "" {
		rpathKey2xpathKey(&xpathB, snode, &keystr)
	}
	return xpathB.String(), nil
}

func InstallRouteRPC(app *fiber.App, rc *RESTCtrl) error {
	app.Group("/restconf/operations/", func(c *fiber.Ctx) error {
		method := c.Method()
		uri := c.Path()
		if method != "POST" {
			return rc.ResponseError(c, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagOperationFailed, fmt.Errorf("use HTTP POST instead of %s for restconf rpc", c.Method()))
		}
		rc.Lock()
		defer rc.Unlock()
		rpcname := uri[len("/restconf/operations/"):]
		schema := rc.schemaOperations.GetSchema(rpcname)
		if schema == nil {
			return rc.ResponseError(c, fiber.StatusNotFound, ETypeApplication, ETagUnknownElement,
				fmt.Errorf("unable to identify rpc %s", rpcname))
		}
		respctrl := &respdata{nodes: []yangtree.DataNode{rc.DataRoot}}
		log.Println(rc.schemaOperations, schema)
		if schema.HasRPCInput() {
			log.Println(string(c.Body()))
			rpc, err := yangtree.New(schema)
			if err != nil {
				return rc.ResponseError(c, fiber.StatusInternalServerError, ETypeApplication,
					ETagOperationFailed, fmt.Errorf("unable to load the schema of the rpc %v: %v", schema, err))
			}
			contentType := string(c.Request().Header.ContentType())
			switch contentType {
			case "text/json", "application/json", "application/yang-data+json":
				err = yangtree.UnmarshalJSON(rpc, c.Body())
			case "text/yaml", "application/yaml", "application/yang-data+yaml":
				err = yangtree.UnmarshalYAML(rpc, c.Body())
			case "text/xml", "application/xml", "application/yang-data+xml":
				err = yangtree.UnmarshalXML(rpc, c.Body())
			default:
				rc.SetError(c, respctrl, fiber.StatusNotImplemented, ETypeProtocol,
					ETagInvalidValue, errors.New("not supported Content-Type"))
			}
			if err != nil {
				return rc.SetError(c, respctrl, fiber.StatusBadRequest,
					ETypeApplication, ETagMarlformedMessage,
					fmt.Errorf("parsing rpc failed: %v", err))
			}
			return nil
		}
		return nil
	})
	return nil
}

func InstallRouteData(app *fiber.App, rc *RESTCtrl) error {
	app.Group("/restconf/data", func(c *fiber.Ctx) error {
		method := c.Method()
		uri := c.Path()[len("/restconf/data"):]
		switch method {
		case "GET":
			rc.RLock()
			defer rc.RUnlock()
		default:
			rc.Lock()
			defer rc.Unlock()
		}

		// requestid := c.GetRespHeader("X-Request-Id")
		respdata := &respdata{nodes: []yangtree.DataNode{rc.DataRoot}}

		switch method {
		case "GET":
			xpath, err := RPath2XPath(rc.schemaData, &uri)
			if err != nil {
				rc.SetError(c, respdata, fiber.StatusInternalServerError,
					ETypeApplication, ETagOperationFailed, err)
				break
			}
			found, err := yangtree.Find(rc.DataRoot, xpath)
			if err != nil {
				rc.SetError(c, respdata, fiber.StatusInternalServerError,
					ETypeApplication, ETagOperationFailed, err)
				break
			}
			if len(found) == 0 {
				rc.SetError(c, respdata, fiber.StatusNotFound, ETypeApplication, ETagDataMissing,
					errors.New("unable to identify the requested resource"))
				break
			}
			respdata.nodes = found
		default:
			rc.SetError(c, respdata, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagOperationNotSupported, fmt.Errorf("HTTP %s not implemented yet", c.Method()))
		}
		// delete(rc.respdata, requestid)
		return rc.Response(c, respdata)
	})
	return nil
}

func InstallRouteRESTCONF(app *fiber.App, rc *RESTCtrl) error {
	empty, err := yangtree.NewWithValue(rc.schemaRESTCONF,
		map[interface{}]interface{}{
			"data":                 map[interface{}]interface{}{},
			"operations":           nil,
			"yang-library-version": rc.yangLibVersion,
		})
	if err != nil {
		log.Fatalf("restconf: %v", err)
	}
	app.All("/restconf", func(c *fiber.Ctx) error {
		respdata := &respdata{nodes: []yangtree.DataNode{empty}}
		switch c.Method() {
		case "GET":
		default:
			rc.SetError(c, respdata, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s", c.Method()))
		}
		return rc.Response(c, respdata)
	})
	app.All("/restconf/yang-library-version", func(c *fiber.Ctx) error {
		respdata := &respdata{nodes: []yangtree.DataNode{empty.Get("yang-library-version")}}
		switch c.Method() {
		case "GET":
		default:
			rc.SetError(c, respdata, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s", c.Method()))
		}
		return rc.Response(c, respdata)
	})
	return nil
}

// register restconf host-meta info.
func InstallRouteHostMeta(app *fiber.App, rc *RESTCtrl) error {
	app.All("/.well-known/host-meta", func(c *fiber.Ctx) error {
		switch c.Method() {
		case "GET":
			c.Links(fmt.Sprint(c.BaseURL() + "/restconf"))
			hdr := &(c.Response().Header)
			hdr.Add("Content-Type", "application/xrd+xml")
			hostmeta :=
				`<XRD xmlns='http://docs.oasis-open.org/ns/xri/xrd-1.0'>
	<Link rel='restconf' href='%s'/>
</XRD>`
			fmt.Fprintf(c, hostmeta, "/restconf")
			return nil
		default:
			respdata := &respdata{}
			rc.SetError(c, respdata, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s to get host-meta", c.Method()))
			return rc.Response(c, respdata)
		}
	})
	return nil
}

// register schema path retrieval
func InstallRouteSchemaPath(app *fiber.App, rc *RESTCtrl) error {
	app.All("/.schema", func(c *fiber.Ctx) error {
		switch c.Method() {
		case "GET":
			hdr := &(c.Response().Header)
			hdr.Add("Content-Type", "application/yang-data+json")
			fmt.Fprintf(c, "[\n")
			allschema := yangtree.CollectSchemaEntries(rc.schemaRESTCONF, true)
			for i := 0; i < len(allschema)-1; i++ {
				fmt.Fprintf(c, " \"%s\",\n", yangtree.GeneratePath(allschema[i], false, false))
			}
			fmt.Fprintf(c, " \"%s\"\n", yangtree.GeneratePath(allschema[len(allschema)-1], false, false))
			fmt.Fprintf(c, "]")
			return nil
		default:
			respdata := &respdata{}
			rc.SetError(c, respdata, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s to get host-meta", c.Method()))
			return rc.Response(c, respdata)
		}
	})
	return nil
}

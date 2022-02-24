package main

import (
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
		if c.Method() != "POST" {
			return NewError(rc, fiber.StatusMethodNotAllowed, ETypeTransport,
				ETagAccessDenied, c.Path(), "HTTP POST only allowed for rpc")
		}
		rc.Lock()
		defer rc.Unlock()
		rpcname := c.Path()[len("/restconf/operations/"):]
		schema := rc.schemaOperations.GetSchema(rpcname)
		if schema == nil {
			return NewError(rc, fiber.StatusNotFound, ETypeProtocol, ETagUnknownElement,
				c.Path(), fmt.Errorf("unable to identify rpc %s", rpcname))
		}

		rpc, err := yangtree.New(schema)
		if err != nil {
			return NewError(rc, fiber.StatusInternalServerError, ETypeProtocol,
				ETagOperationFailed, c.Path(), err)
		}

		if schema.HasRPCInput() {
			// log.Println(string(c.Body()))
			contentType := string(c.Request().Header.ContentType())
			switch contentType {
			case "text/json", "application/json", "application/yang-data+json":
				err = yangtree.UnmarshalJSON(rpc, c.Body())
			case "text/yaml", "application/yaml", "application/yang-data+yaml":
				err = yangtree.UnmarshalYAML(rpc, c.Body())
			case "text/xml", "application/xml", "application/yang-data+xml":
				err = yangtree.UnmarshalXML(rpc, c.Body())
			default:
				return NewError(rc, fiber.StatusUnsupportedMediaType, ETypeTransport,
					ETagInvalidValue, c.Path(), "not supported Content-Type in request header")
			}
			// log.Println(rpc.Values()...)
			if err != nil {
				return NewError(rc, fiber.StatusBadRequest, ETypeApplication,
					ETagMarlformedMessage, c.Path(), fmt.Sprintf("parsing error: %v", err))
			}
		}
		// invoke user-callback interface
		// check the result of the user-callback

		if schema.HasRPCOutput() {
			if output := rpc.Get("output"); output != nil {
				return rc.Response(c, &RespData{Nodes: []yangtree.DataNode{output}})
			}
		}

		// If the RPC operation is invoked without errors and if the "rpc" or
		// "action" statement has no "output" section, the response message
		// MUST NOT include a message-body and MUST send a "204 No Content"
		// status-line instead.
		return rc.Response(c, &RespData{Status: fiber.StatusNoContent})
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

		switch method {
		case "GET":
			xpath, err := RPath2XPath(rc.schemaData, &uri)
			if err != nil {
				return NewError(rc, fiber.StatusInternalServerError, ETypeApplication,
					ETagBadElement, c.Path(), err)
			}
			found, err := yangtree.Find(rc.DataRoot, xpath)
			if err != nil {
				return NewError(rc, fiber.StatusInternalServerError, ETypeApplication,
					ETagOperationFailed, c.Path(), err)
			}
			if len(found) == 0 {
				return NewError(rc, fiber.StatusNotFound, ETypeApplication,
					ETagDataMissing, c.Path(), "unable to find the requested resource")
			}
			rdata := &RespData{Nodes: found}
			return rc.Response(c, rdata)
		default:
			return NewError(rc, fiber.StatusMethodNotAllowed, ETypeProtocol, ETagOperationFailed,
				uri, fmt.Errorf("HTTP %s not implemented yet", method))
		}
	})
	return nil
}

func InstallRouteRESTCONF(app *fiber.App, rc *RESTCtrl) error {
	// Check Request Validation
	app.Group("/restconf", func(c *fiber.Ctx) error {
		c.Set("Server", "open-restconf")
		c.Set("Cache-Control", "no-cache")
		switch c.Method() {
		case "PUT":
		case "GET":
		default:

		}

		accepts := c.Accepts("*/*", "text/json", "text/yaml", "text/xml",
			"application/xml", "application/json", "application/yaml",
			"application/yang-data+xml", "application/yang-data+json", "application/yang-data+yaml")
		switch {
		case accepts == "*/*": // if all types are allowed
			c.Set("Content-Type", "application/yang-data+xml")
		case strings.HasSuffix(accepts, "xml"):
			c.Set("Content-Type", accepts)
		case strings.HasSuffix(accepts, "json"):
			c.Set("Content-Type", accepts)
		case strings.HasSuffix(accepts, "yaml"):
			c.Set("Content-Type", accepts)
		default:
			return NewError(rc, fiber.StatusNotAcceptable, ETypeTransport,
				ETagInvalidValue, c.Path(), "unsupported Accepts (Content-Type) header")
		}
		if err := c.Next(); err != nil {
			return err
		}
		// send an error if the resource not found.
		if c.Response().StatusCode() == fiber.StatusNotFound {
			return NewError(rc, fiber.StatusNotFound, ETypeApplication,
				ETagDataMissing, c.Path(), "resource not found")
		}
		return nil
	})
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
		switch c.Method() {
		case "GET":
		default:
			return NewError(rc, fiber.StatusMethodNotAllowed, ETypeTransport,
				ETagAccessDenied, c.Path(), "HTTP GET only allowed for the path")
		}
		return rc.Response(c, &RespData{Nodes: []yangtree.DataNode{empty}})
	})
	app.All("/restconf/yang-library-version", func(c *fiber.Ctx) error {
		switch c.Method() {
		case "GET":
		default:
			return NewError(rc, fiber.StatusMethodNotAllowed, ETypeTransport,
				ETagAccessDenied, c.Path(), "HTTP GET only allowed for the path")
		}
		return rc.Response(c, &RespData{Nodes: []yangtree.DataNode{empty.Get("yang-library-version")}})
	})

	if err := InstallRouteData(app, rc); err != nil {
		log.Fatalf("restconf: %v", err)
	}
	if err := InstallRouteRPC(app, rc); err != nil {
		log.Fatalf("restconf: %v", err)
	}
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
			return NewError(rc, fiber.StatusMethodNotAllowed, ETypeTransport,
				ETagAccessDenied, c.Path(), "HTTP GET only allowed for the path")
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
			return NewError(rc, fiber.StatusMethodNotAllowed, ETypeTransport,
				ETagAccessDenied, c.Path(), "HTTP GET only allowed for the path")
		}
	})
	return nil
}

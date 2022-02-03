package main

import (
	"errors"
	"fmt"
	"log"
	"strings"

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

func InstallRouteRPC(router fiber.Router, schema *yangtree.SchemaNode, rc *RESTCtrl) error {
	routePath, _ := GenerateRoutePath(schema, false)
	for i := range routePath {
		router.All(routePath[i], func(c *fiber.Ctx) error {
			respctrl := rc.getRespCtrl(c)
			if c.Method() != "POST" {
				return rc.SetError(c, respctrl, fiber.StatusMethodNotAllowed, ETypeProtocol,
					ETagOperationFailed, fmt.Errorf("use HTTP POST instead of %s for restconf rpc", c.Method()))
			}
			if schema.HasRPCInput() {
				log.Println(string(c.Body()))
				rpc, err := yangtree.New(schema)
				if err != nil {
					return rc.SetError(c, respctrl, fiber.StatusInternalServerError, ETypeApplication,
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
	}
	return nil
}

func InstallDirectoryRoute(router fiber.Router, schema *yangtree.SchemaNode, rc *RESTCtrl) error {
	routePath, searchPath := GenerateRoutePath(schema, false)
	for i := range routePath {
		dirgroup := router.Group(routePath[i], func(c *fiber.Ctx) error {
			respctrl := rc.getRespCtrl(c)
			var p string
			pname := c.Route().Params
			if len(pname) > 0 {
				pdata := make([]interface{}, len(pname))
				for j := range pname {
					pdata[j] = c.Params(pname[j])
				}
				p = fmt.Sprintf(searchPath[i], pdata...)
				if schema.IsList() {
					respctrl.groupSearch = false
				}
			} else {
				p = searchPath[0]
				if schema.IsList() {
					respctrl.groupSearch = true
				}
			}
			var found []yangtree.DataNode
			for j := range respctrl.nodes {
				if schema.Name == respctrl.nodes[j].Name() {
					// select matched nodes with the params.
					if p == respctrl.nodes[j].ID() {
						found = append(found, respctrl.nodes[j])
					}
				} else {
					n, err := yangtree.Find(respctrl.nodes[j], p)
					if err != nil {
						return rc.SetError(c, respctrl, fiber.StatusInternalServerError,
							ETypeApplication, ETagOperationFailed, err)
					}
					found = append(found, n...)
				}
			}
			if len(found) == 0 {
				return rc.SetError(c, respctrl, fiber.StatusNotFound, ETypeApplication, ETagDataMissing, nil)
			}
			respctrl.nodes = found
			log.Println(" => ", c.Path(), p, c.Route().Params, "RESULT", respctrl.nodes)
			return c.Next()
		})
		router.All(routePath[i], func(c *fiber.Ctx) error {
			switch c.Method() {
			case "GET":
				// return without error because the target nodes are already found.
				return nil
			default:
				return rc.SetError(c, nil, fiber.StatusMethodNotAllowed, ETypeProtocol,
					ETagOperationNotSupported, fmt.Errorf("HTTP %s not implemented yet", c.Method()))
			}
		})
		for j := range schema.Children {
			if err := InstallRoute(dirgroup, schema.Children[j], rc); err != nil {
				return err
			}
		}
	}
	return nil
}

func InstallNoneDirectoryRoute(router fiber.Router, schema *yangtree.SchemaNode, rc *RESTCtrl) error {
	routePath, searchPath := GenerateRoutePath(schema, false)
	for i := range routePath {
		router.All(routePath[i], func(c *fiber.Ctx) error {
			respctrl := rc.getRespCtrl(c)
			switch c.Method() {
			case "GET":
				var p string
				pname := c.Route().Params
				if len(pname) > 0 {
					pdata := make([]interface{}, len(pname))
					for j := range pname {
						pdata[j] = c.Params(pname[j])
					}
					p = fmt.Sprintf(searchPath[i], pdata...)
				} else {
					p = searchPath[i]
					if schema.IsListable() {
						respctrl.groupSearch = true
					}
				}
				var node []yangtree.DataNode
				for j := range respctrl.nodes {
					n, err := yangtree.Find(respctrl.nodes[j], p)
					if err != nil {
						return rc.SetError(c, nil, fiber.StatusInternalServerError,
							ETypeApplication, ETagOperationFailed, err)
					}
					node = append(node, n...)
				}
				if len(node) == 0 {
					return rc.SetError(c, nil, fiber.StatusNotFound, ETypeApplication, ETagDataMissing, nil)
				}
				respctrl.nodes = node
				return nil
			default:
				return rc.SetError(c, nil, fiber.StatusMethodNotAllowed, ETypeProtocol,
					ETagOperationNotSupported, fmt.Errorf("HTTP %s not implemented yet", c.Method()))
			}
		})
	}
	return nil
}

func InstallRoute(router fiber.Router, schema *yangtree.SchemaNode, rc *RESTCtrl) error {
	log.Println("InstallRoute", schema.Path())
	switch {
	case schema.IsRPC():
		return InstallRouteRPC(router, schema, rc)
	case schema.IsDir():
		return InstallDirectoryRoute(router, schema, rc)
	default:
		return InstallNoneDirectoryRoute(router, schema, rc)
	}
}

func InstallRouteData(app *fiber.App, rc *RESTCtrl) error {
	top := app.Group("/restconf/data", func(c *fiber.Ctx) error {
		log.Println(c.Method(), c.Path())
		switch c.Method() {
		case "GET":
			rc.RLock()
			defer rc.RUnlock()
		default:
			rc.Lock()
			defer rc.Unlock()
		}

		requestid := c.GetRespHeader("X-Request-Id")
		respctrl := &RespCtrl{nodes: []yangtree.DataNode{rc.DataRoot}}
		rc.RespCtrl[requestid] = respctrl
		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			rc.SetError(c, respctrl, status, ETypeApplication, ETagOperationFailed, err)
		} else if len(respctrl.errors) == 0 {
			switch status {
			case fiber.StatusOK:
				break
			case fiber.StatusNotFound:
				rc.SetError(c, respctrl, status, ETypeApplication, ETagUnknownElement,
					errors.New("unable to identify the requested resource"))
			default:
				rc.SetError(c, respctrl, status, ETypeApplication, ETagOperationFailed,
					errors.New("unable to identify the requested resource"))
			}
		}
		delete(rc.RespCtrl, requestid)
		return rc.Response(c, respctrl)
	})
	app.Get("/restconf/data", func(c *fiber.Ctx) error {
		return nil
	})
	for i := range rc.schemaData.Children {
		if err := InstallRoute(top, rc.schemaData.Children[i], rc); err != nil {
			log.Fatalf("restconf: %v", err)
		}
	}
	return nil
}

func InstallRouteRoot(app *fiber.App, rc *RESTCtrl) error {
	// register restconf root route
	emptyRoot, err := yangtree.NewWithValue(rc.schemaRESTCONF,
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
			// FIXME - update Response
			b, err := yangtree.MarshalJSONIndent(emptyRoot, "", " ")
			if err != nil {
				// rc.SetError(c, fiber.StatusInternalServerError, ETypeRPC, ETagOperationFailed, err)
			}
			return c.Send(b)
		default:
			return rc.SetError(c, nil, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s", c.Method()))
		}
	})
	app.All("/restconf/yang-library-version", func(c *fiber.Ctx) error {
		switch c.Method() {
		case "GET":
			// FIXME - update Response
			b, err := yangtree.MarshalJSONIndent(emptyRoot.Get("yang-library-version"), "", " ", yangtree.RepresentItself{})
			if err != nil {
				// rc.SetError(c, fiber.StatusInternalServerError, ETypeRPC, ETagOperationFailed, err)
			}
			return c.Send(b)
		default:
			return rc.SetError(c, nil, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s", c.Method()))
		}
	})

	if err := InstallRouteData(app, rc); err != nil {
		log.Fatalf("restconf: %v", err)
	}
	return nil
}

func InstallRouteHostMeta(app *fiber.App, rc *RESTCtrl) error {
	// register restconf host-meta info.

	app.All("/.well-known/host-meta", func(c *fiber.Ctx) error {
		switch c.Method() {
		case "GET":
			// FIXME - add a link for the restconf access point
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
			return rc.SetError(c, nil, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s to get host-meta", c.Method()))
		}
	})
	return nil
}

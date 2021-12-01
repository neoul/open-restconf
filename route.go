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

func InstallRouteRPC(router fiber.Router, schema *yangtree.SchemaNode, rctrl *RestconfCtrl) error {
	routePath, _ := GenerateRoutePath(schema, false)
	for i := range routePath {
		router.All(routePath[i], func(c *fiber.Ctx) error {
			if c.Method() != "POST" {
				return rctrl.SetError(c, fiber.StatusMethodNotAllowed, ETypeProtocol,
					ETagOperationFailed, fmt.Errorf("use HTTP POST instead of %s for restconf rpc", c.Method()))
			}
			if schema.HasRPCInput() {
				log.Println(string(c.Body()))
				rpc, err := yangtree.New(schema)
				if err != nil {
					return rctrl.SetError(c, fiber.StatusInternalServerError, ETypeApplication,
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
					rctrl.SetError(c, fiber.StatusNotImplemented, ETypeProtocol,
						ETagInvalidValue, errors.New("not supported Content-Type"))
				}
				if err != nil {
					return rctrl.SetError(c, fiber.StatusBadRequest,
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

func InstallDirectoryRoute(router fiber.Router, schema *yangtree.SchemaNode, rctrl *RestconfCtrl) error {
	routePath, searchPath := GenerateRoutePath(schema, false)
	for i := range routePath {
		dirgroup := router.Group(routePath[i], func(c *fiber.Ctx) error {
			var p string
			pname := c.Route().Params
			if len(pname) > 0 {
				pdata := make([]interface{}, len(pname))
				for j := range pname {
					pdata[j] = c.Params(pname[j])
				}
				p = fmt.Sprintf(searchPath[i], pdata...)
				if schema.IsList() {
					rctrl.isGroupSearch = false
				}
			} else {
				p = searchPath[0]
				if schema.IsList() {
					rctrl.isGroupSearch = true
				}
			}
			var found []yangtree.DataNode
			for j := range rctrl.curnode {
				if schema.Name == rctrl.curnode[j].Name() {
					// select matched nodes with the params.
					if p == rctrl.curnode[j].ID() {
						found = append(found, rctrl.curnode[j])
					}
				} else {
					n, err := yangtree.Find(rctrl.curnode[j], p)
					if err != nil {
						return rctrl.SetError(c, fiber.StatusInternalServerError,
							ETypeApplication, ETagOperationFailed, err)
					}
					found = append(found, n...)
				}
			}
			if len(found) == 0 {
				return rctrl.SetError(c, fiber.StatusNotFound, ETypeApplication, ETagDataMissing, nil)
			}
			rctrl.curnode = found
			log.Println(" => ", c.Path(), p, c.Route().Params, "RESULT", rctrl.curnode)
			return c.Next()
		})
		router.All(routePath[i], func(c *fiber.Ctx) error {
			switch c.Method() {
			case "GET":
				// return without error because the target nodes are already found.
				return nil
			default:
				return rctrl.SetError(c, fiber.StatusMethodNotAllowed, ETypeProtocol,
					ETagOperationNotSupported, fmt.Errorf("HTTP %s not implemented yet", c.Method()))
			}
		})
		for j := range schema.Children {
			if err := InstallRoute(dirgroup, schema.Children[j], rctrl); err != nil {
				return err
			}
		}
	}
	return nil
}

func InstallNoneDirectoryRoute(router fiber.Router, schema *yangtree.SchemaNode, rctrl *RestconfCtrl) error {
	routePath, searchPath := GenerateRoutePath(schema, false)
	for i := range routePath {
		router.All(routePath[i], func(c *fiber.Ctx) error {
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
						rctrl.isGroupSearch = true
					}
				}
				var node []yangtree.DataNode
				for j := range rctrl.curnode {
					n, err := yangtree.Find(rctrl.curnode[j], p)
					if err != nil {
						return rctrl.SetError(c, fiber.StatusInternalServerError,
							ETypeApplication, ETagOperationFailed, err)
					}
					node = append(node, n...)
				}
				if len(node) == 0 {
					return rctrl.SetError(c, fiber.StatusNotFound, ETypeApplication, ETagDataMissing, nil)
				}
				rctrl.curnode = node
				return nil
			default:
				return rctrl.SetError(c, fiber.StatusMethodNotAllowed, ETypeProtocol,
					ETagOperationNotSupported, fmt.Errorf("HTTP %s not implemented yet", c.Method()))
			}
		})
	}
	return nil
}

func InstallRoute(router fiber.Router, schema *yangtree.SchemaNode, rctrl *RestconfCtrl) error {
	log.Println("InstallRoute", schema.Path())
	switch {
	case schema.IsRPC():
		return InstallRouteRPC(router, schema, rctrl)
	case schema.IsDir():
		return InstallDirectoryRoute(router, schema, rctrl)
	default:
		return InstallNoneDirectoryRoute(router, schema, rctrl)
	}
}

func InstallRouteRoot(app *fiber.App, rctrl *RestconfCtrl) error {
	top := app.Group("/restconf", func(c *fiber.Ctx) error {
		log.Println(c.Method(), c.Path())
		rctrl.Lock()
		defer rctrl.Unlock()
		rctrl.isGroupSearch = false
		rctrl.curnode = []yangtree.DataNode{rctrl.DataNode}
		if len(rctrl.errors) > 0 {
			rctrl.errors = rctrl.errors[:0]
		}

		err := c.Next()
		status := c.Response().StatusCode()
		if err != nil {
			rctrl.SetError(c, status, ETypeApplication, ETagOperationFailed, err)
		} else if len(rctrl.errors) == 0 {
			switch status {
			case fiber.StatusOK:
				break
			case fiber.StatusNotFound:
				rctrl.SetError(c, status, ETypeApplication, ETagUnknownElement,
					errors.New("unable to identify the requested resource"))
			default:
				rctrl.SetError(c, status, ETypeApplication, ETagOperationFailed,
					errors.New("unable to identify the requested resource"))
			}
		}
		return rctrl.Response(c)
	})
	app.Get("/restconf", func(c *fiber.Ctx) error {
		return nil
	})
	for i := range restconfSchema.Children {
		if err := InstallRoute(top, restconfSchema.Children[i], rctrl); err != nil {
			log.Fatalf("restconf: %v", err)
		}
	}
	return nil
}

func InstallRouteHostMeta(app *fiber.App, rctrl *RestconfCtrl) error {
	// register restconf host-meta info.
	app.All("/.well-known/host-meta", func(c *fiber.Ctx) error {
		switch c.Method() {
		case "GET":
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
		default:
			return rctrl.SetError(c, fiber.StatusMethodNotAllowed, ETypeProtocol,
				ETagResourceDenied, fmt.Errorf("use HTTP GET instead of %s to get host-meta", c.Method()))
		}
	})
	return nil
}

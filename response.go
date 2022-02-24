package main

import (
	"strings"

	"github.com/gofiber/fiber"

	"github.com/neoul/yangtree"
)

//
type RespData struct {
	Nodes       []yangtree.DataNode
	groupSearch bool // true if searching multipleNnodes
	status      int  // HTTP response status
}

func (rc *RESTCtrl) Response(c *fiber.Ctx, rdata *RespData) error {
	// Response content priority
	// 1. Header error
	// 2. Content error
	// 3. No error
	if len(rdata.Nodes) == 0 {
		return NewError(rc, fiber.StatusNotFound, ETypeApplication,
			ETagOperationFailed, c.Path(), "resource not found")
	}

	c.Set("Server", "open-restconf")
	c.Set("Cache-Control", "no-cache")

	marshal := yangtree.MarshalXMLIndent
	accepts := c.Accepts("*/*", "text/json", "text/yaml", "text/xml",
		"application/xml", "application/json", "application/yaml",
		"application/yang-data+xml", "application/yang-data+json", "application/yang-data+yaml")
	switch {
	case accepts == "*/*": // if all types are allowed
		c.Set("Content-Type", "application/yang-data+xml")
		marshal = yangtree.MarshalXMLIndent
	case strings.HasSuffix(accepts, "xml"):
		c.Set("Content-Type", accepts)
		marshal = yangtree.MarshalXMLIndent
	case strings.HasSuffix(accepts, "json"):
		c.Set("Content-Type", accepts)
		marshal = yangtree.MarshalJSONIndent
	case strings.HasSuffix(accepts, "yaml"):
		c.Set("Content-Type", accepts)
		marshal = yangtree.MarshalYAMLIndent
	default:
		c.Set("Content-Type", "application/yang-data+xml")
		marshal = yangtree.MarshalXMLIndent
	}
	switch c.Method() {
	case "GET":
		var err error
		var node yangtree.DataNode
		if rdata.groupSearch || len(rdata.Nodes) > 1 {
			node, err = yangtree.ConvertToGroup(rdata.Nodes[0].Schema(), rdata.Nodes)
			if err != nil {
				// StatusPreconditionFailed - for GET or HEAD when If-Unmodified-Since or If-None-Match headers is not fulfilled.
				return NewError(rc, fiber.StatusInternalServerError, ETypeProtocol,
					ETagOperationFailed, c.Path(), err)
			}
		} else {
			node = rdata.Nodes[0]
		}
		b, err := marshal(node, "", " ", yangtree.RepresentItself{})
		if err != nil {
			return NewError(rc, fiber.StatusInternalServerError, ETypeProtocol,
				ETagOperationFailed, c.Path(), err)
		}
		return c.Send(b)
	case "POST":
	}
	return nil
}

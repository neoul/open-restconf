package main

import (
	"strings"

	"github.com/gofiber/fiber"

	"github.com/neoul/yangtree"
)

//
type RespData struct {
	Nodes   []yangtree.DataNode
	isGroup bool // true if searching multipleNnodes
	Status  int  // HTTP response status
}

func (rc *RESTCtrl) Response(c *fiber.Ctx, rdata *RespData) error {
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
	case "GET": // netconf get, get-config
		if len(rdata.Nodes) == 0 {
			return NewError(rc, fiber.StatusNotFound, ETypeApplication,
				ETagDataMissing, c.Path(), "resource not found")
		}
		var err error
		var b []byte
		var node yangtree.DataNode
		if rdata.isGroup || len(rdata.Nodes) > 1 {
			node, err = yangtree.ConvertToGroup(rdata.Nodes[0].Schema(), rdata.Nodes)
			if err != nil {
				// StatusPreconditionFailed - for GET or HEAD
				// when If-Unmodified-Since or If-None-Match headers is not fulfilled.
				return NewError(rc, fiber.StatusInternalServerError, ETypeApplication,
					ETagOperationFailed, c.Path(), err)
			}
		} else {
			node = rdata.Nodes[0]
		}
		b, err = marshal(node, "", " ", yangtree.RepresentItself{})
		if err != nil {
			return NewError(rc, fiber.StatusInternalServerError, ETypeApplication,
				ETagOperationFailed, c.Path(), err)
		}
		if rdata.Status != 0 {
			c.Status(rdata.Status)
		}
		return c.Send(b)
	case "POST": // netconf rpc
		if rdata.Status != 0 {
			c.Status(rdata.Status)
		}
	}
	return nil
}

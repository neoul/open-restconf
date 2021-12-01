package main

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber"
	"github.com/neoul/yangtree"
)

func (rctrl *RestconfCtrl) Response(c *fiber.Ctx) error {
	var err error
	var enode yangtree.DataNode
	var marshal func(node yangtree.DataNode, prefix, indent string, option ...yangtree.Option) ([]byte, error)

	c.Set("Server", "open-restconf")
	c.Set("Cache-Control", "no-cache")

	marshal = yangtree.MarshalXMLIndent
	accepts := c.Accepts("text/json", "text/yaml", "text/xml",
		"application/xml", "application/json", "application/yaml",
		"application/yang-data+xml", "application/yang-data+json", "application/yang-data+yaml")
	switch {
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
		rctrl.SetError(c, fiber.StatusNotAcceptable, ETypeProtocol,
			ETagInvalidValue, errors.New("not supported Content-Type"))
	}

	if len(rctrl.errors) == 0 {
		switch c.Method() {
		case "GET":
			var node yangtree.DataNode
			if rctrl.isGroupSearch {
				node, err = yangtree.ConvertToGroup(rctrl.curnode[0].Schema(), rctrl.curnode)
				if err != nil {
					// StatusPreconditionFailed - for GET or HEAD when If-Unmodified-Since or If-None-Match headers is not fulfilled.
					rctrl.SetError(c, fiber.StatusInternalServerError, ETypeApplication, ETagOperationFailed, err)
					break
				}
			} else {
				node = rctrl.curnode[0]
			}
			b, err := marshal(node, "", " ")
			if err != nil {
				rctrl.SetError(c, fiber.StatusInternalServerError, ETypeRPC, ETagOperationFailed, err)
				break
			}
			return c.Send(b)
		case "POST":
		}
	}

	if len(rctrl.errors) > 0 {
		if errorSchema == nil {
			log.Fatalf("restconf: errors schema not loaded")
		}
		enode, err = yangtree.New(errorSchema)
		if err != nil {
			log.Fatalf("restconf: errors/error schema not loaded")
		}
		for i := range rctrl.errors {
			if _, err := enode.Insert(rctrl.errors[i], nil); err != nil {
				log.Fatalf("restconf: fault in error report: %v", err)
			}
		}

		b, err := marshal(enode, "", " ")
		if err != nil {
			log.Fatalf("restconf: fault in error report: %v", err)
		}
		return c.Send(b)
	}
	return nil
}

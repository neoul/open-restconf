package main

import (
	"errors"
	"log"
	"strings"

	"github.com/gofiber/fiber"

	"github.com/neoul/yangtree"
)

func (rc *RESTCtrl) Response(c *fiber.Ctx, respctrl *RespCtrl) error {

	var err error
	var enode yangtree.DataNode
	var marshal func(node yangtree.DataNode, prefix, indent string, option ...yangtree.Option) ([]byte, error)

	c.Set("Server", "open-restconf")
	c.Set("Cache-Control", "no-cache")

	marshal = yangtree.MarshalXMLIndent
	accepts := c.Accepts("*/*", "text/json", "text/yaml", "text/xml",
		"application/xml", "application/json", "application/yaml",
		"application/yang-data+xml", "application/yang-data+json", "application/yang-data+yaml")
	switch {
	case accepts == "*/*":
		fallthrough
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
		rc.SetError(c, respctrl, fiber.StatusNotAcceptable, ETypeProtocol,
			ETagInvalidValue, errors.New("not supported Content-Type"))
	}

	if len(respctrl.errors) == 0 {
		switch c.Method() {
		case "GET":
			var node yangtree.DataNode
			if respctrl.groupSearch {
				node, err = yangtree.ConvertToGroup(respctrl.nodes[0].Schema(), respctrl.nodes)
				if err != nil {
					// StatusPreconditionFailed - for GET or HEAD when If-Unmodified-Since or If-None-Match headers is not fulfilled.
					rc.SetError(c, respctrl, fiber.StatusInternalServerError,
						ETypeApplication, ETagOperationFailed, err)
					break
				}
			} else {
				node = respctrl.nodes[0]
			}
			b, err := marshal(node, "", " ")
			if err != nil {
				rc.SetError(c, respctrl, fiber.StatusInternalServerError, ETypeRPC, ETagOperationFailed, err)
				break
			}
			return c.Send(b)
		case "POST":
		}
	}

	if len(respctrl.errors) > 0 {
		enode, err = yangtree.New(rc.schemaErrors)
		if err != nil {
			log.Fatalf("restconf: errors/error schema not loaded")
		}
		for i := range respctrl.errors {
			if _, err := enode.Insert(respctrl.errors[i], nil); err != nil {
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

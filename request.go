package main

import (
	"github.com/gofiber/fiber"
	"github.com/neoul/yangtree"
)

func (rc *RESTCtrl) Request(c *fiber.Ctx, rdata *RespData, schema *yangtree.SchemaNode) error {
	// rpc, err := yangtree.New(schema)
	// if err != nil {
	// 	return rc.ResponseError(c, fiber.StatusInternalServerError, ETypeApplication,
	// 		ETagOperationFailed, fmt.Errorf("unable to load the schema of the rpc %v: %v", schema, err))
	// }

	// contentType := string(c.Request().Header.ContentType())
	// switch {
	// case strings.HasSuffix(contentType, "json"):
	// 	err = yangtree.UnmarshalJSON(rpc, c.Body())
	// case strings.HasSuffix(contentType, "yaml"):
	// 	err = yangtree.UnmarshalYAML(rpc, c.Body())
	// case strings.HasSuffix(contentType, "xml"):
	// 	err = yangtree.UnmarshalXML(rpc, c.Body())
	// default:
	// 	return rc.SetError(c, rdata, fiber.StatusNotImplemented, ETypeProtocol,
	// 		ETagInvalidValue, errors.New("not supported Content-Type"))
	// }
	// if err != nil {
	// 	return rc.SetError(c, rdata, fiber.StatusBadRequest,
	// 		ETypeApplication, ETagMarlformedMessage,
	// 		fmt.Errorf("parsing rpc failed: %v", err))
	// }
	return nil
}

package main

import (
	"log"
	"strings"

	"github.com/gofiber/fiber"
	"github.com/neoul/yangtree"
)

// RFC6248 - 4.3. <rpc-error> Element
//
// error-type:  Defines the conceptual layer that the error occurred.
// Enumeration.  One of:
//  *  transport (layer: Secure Transport)
//  *  rpc (layer: Messages)
//  *  protocol (layer: Operations)
//  *  application (layer: Content)
type ErrorType int

const (
	ETypeApplication ErrorType = iota // error related to schema or data node
	ETypeProtocol                     // error in rpc (including user-defined rpc)
	ETypeRPC                          // error in message Format/char-types
	ETypeTransport                    // error in HTTP/TLS
)

func (et ErrorType) String() string {
	switch et {
	case ETypeApplication:
		return "application"
	case ETypeProtocol:
		return "protocol"
	case ETypeRPC:
		return "rpc"
	case ETypeTransport:
		return "transport"
	default:
		return "unknown"
	}
}

// NETCONF error (https://datatracker.ietf.org/doc/html/rfc6241#appendix-A)

type ErrorTag int

const (
	ETagInUse ErrorTag = iota
	ETagInvalidValue
	ETagTooBig
	ETagMissingAttribute
	ETagBadAttribute
	ETagUnknownAttribute
	ETagMissingElement
	ETagBadElement
	ETagUnknownElement
	ETagUnknownNamespace
	ETagAccessDenied
	ETagLockDenied
	ETagResourceDenied
	ETagRollbackFailed
	ETagDataExists
	ETagDataMissing
	ETagOperationNotSupported
	ETagOperationFailed
	ETagPartialOperation
	ETagMarlformedMessage
)

func (et ErrorTag) String() string {
	switch et {
	case ETagInUse:
		return "in-use"
	case ETagInvalidValue:
		return "invalid-value"
	case ETagTooBig:
		return "too-big"
	case ETagMissingAttribute:
		return "missing-attribute"
	case ETagBadAttribute:
		return "bad-attribute"
	case ETagUnknownAttribute:
		return "unknown-attribute"
	case ETagMissingElement:
		return "missing-element"
	case ETagBadElement:
		return "bad-element"
	case ETagUnknownElement:
		return "unknown-element"
	case ETagUnknownNamespace:
		return "unknown-namespace"
	case ETagAccessDenied:
		return "access-denied"
	case ETagLockDenied:
		return "lock-denied"
	case ETagResourceDenied:
		return "resource-denied"
	case ETagRollbackFailed:
		return "rollback-failed"
	case ETagDataExists:
		return "data-exists"
	case ETagDataMissing:
		return "data-missing"
	case ETagOperationNotSupported:
		return "operation-not-supported"
	case ETagOperationFailed:
		return "operation-failed"
	case ETagPartialOperation:
		return "partial-operation"
	case ETagMarlformedMessage:
		return "marlformed-message"
	default:
		return "unknown"
	}
}

// Status() returns a HTTP code according to the Tag.
// +-------------------------+------------------+
// | error-tag               | status code      |
// +-------------------------+------------------+
// | in-use                  | 409              |
// | invalid-value           | 400, 404, or 406 |
// | (request) too-big       | 413              |
// | (response) too-big      | 400              |
// | missing-attribute       | 400              |
// | bad-attribute           | 400              |
// | unknown-attribute       | 400              |
// | bad-element             | 400              |
// | unknown-element         | 400              |
// | unknown-namespace       | 400              |
// | access-denied           | 401 or 403       |
// | lock-denied             | 409              |
// | resource-denied         | 409              |
// | rollback-failed         | 500              |
// | data-exists             | 409              |
// | data-missing            | 409              |
// | operation-not-supported | 405 or 501       |
// | operation-failed        | 412 or 500       |
// | partial-operation       | 500              |
// | malformed-message       | 400              |
// +-------------------------+------------------+
// 	Mapping from <error-tag> to Status Code
func (et ErrorTag) Status() int {
	switch et {
	case ETagInUse:
		return fiber.StatusConflict
	case ETagInvalidValue:
		return fiber.StatusBadRequest
		// return fiber.StatusNotFound
		// return fiber.StatusNotAcceptable
	case ETagTooBig:
		return fiber.StatusRequestEntityTooLarge
	case ETagMissingAttribute:
		return fiber.StatusBadRequest
	case ETagBadAttribute:
		return fiber.StatusBadRequest
	case ETagUnknownAttribute:
		return fiber.StatusBadRequest
	case ETagMissingElement:
		return fiber.StatusBadRequest
	case ETagBadElement:
		return fiber.StatusBadRequest
	case ETagUnknownElement:
		return fiber.StatusBadRequest
	case ETagUnknownNamespace:
		return fiber.StatusBadRequest
	case ETagAccessDenied:
		return fiber.StatusUnauthorized
		// return fiber.StatusForbidden
	case ETagLockDenied:
		return fiber.StatusConflict
	case ETagResourceDenied:
		return fiber.StatusConflict
	case ETagRollbackFailed:
		return fiber.StatusInternalServerError
	case ETagDataExists:
		return fiber.StatusConflict
	case ETagDataMissing:
		return fiber.StatusConflict
	case ETagOperationNotSupported:
		// return fiber.StatusMethodNotAllowed
		return fiber.StatusNotImplemented
	case ETagOperationFailed:
		// return fiber.StatusPreconditionFailed
		return fiber.StatusInternalServerError
	case ETagPartialOperation:
		return fiber.StatusInternalServerError
	case ETagMarlformedMessage:
		return fiber.StatusBadRequest
	default:
		return fiber.StatusInternalServerError
	}
}

func errhandler(c *fiber.Ctx, err error) error {
	if e, ok := err.(*RespError); ok {
		return e.Response(c)
	}
	// Status code defaults to 500
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).SendString(err.Error())
}

type RespError struct {
	Errors []yangtree.DataNode
	Code   int // HTTP response status
}

func NewError(rc *RESTCtrl, code int, etyp ErrorType, etag ErrorTag, epath string, emsg interface{}) *RespError {
	re := &RespError{}
	e, err := yangtree.NewWithValue(rc.schemaError,
		map[interface{}]interface{}{
			"error-tag":  etag.String(),
			"error-type": etyp.String(),
			"error-path": epath,
		})
	if err != nil {
		log.Fatalf("restconf: fault in error report: %v", err)
	}
	if fe, ok := emsg.(*fiber.Error); ok {
		err = yangtree.SetValue(e, "error-message", nil, fe.Message)
	} else if s, ok := emsg.(string); ok {
		err = yangtree.SetValue(e, "error-message", nil, s)
	} else if em, ok := emsg.(error); ok {
		err = yangtree.SetValue(e, "error-message", nil, em.Error())
	}
	if err != nil {
		log.Fatalf("restconf: fault in error report: %v", err)
	}
	re.Errors = append(re.Errors, e)
	re.Code = code
	return re
}

func (re *RespError) Add(rc *RESTCtrl, code int, etyp ErrorType, etag ErrorTag, epath string, emsg interface{}) *RespError {
	if re == nil {
		return NewError(rc, code, etyp, etag, epath, emsg)
	}
	e, err := yangtree.NewWithValue(rc.schemaError,
		map[interface{}]interface{}{
			"error-tag":  etag.String(),
			"error-type": etyp.String(),
			"error-path": epath,
		})
	if err != nil {
		log.Fatalf("restconf: fault in error report: %v", err)
	}
	if fe, ok := emsg.(*fiber.Error); ok {
		err = yangtree.SetValue(e, "error-message", nil, fe.Message)
	} else if s, ok := emsg.(string); ok {
		err = yangtree.SetValue(e, "error-message", nil, s)
	} else if em, ok := emsg.(error); ok {
		err = yangtree.SetValue(e, "error-message", nil, em.Error())
	}
	if err != nil {
		log.Fatalf("restconf: fault in error report: %v", err)
	}
	re.Errors = append(re.Errors, e)
	return re
}

func (re *RespError) Error() string {
	if len(re.Errors) > 0 {
		errorsSchema := re.Errors[0].Schema().Parent
		enode, err := yangtree.New(errorsSchema)
		if err != nil {
			log.Fatalf("restconf: errors/error schema not loaded")
		}
		for i := range re.Errors {
			if _, err := enode.Insert(re.Errors[i], nil); err != nil {
				log.Fatalf("restconf: fault in error report: %v", err)
			}
		}

		b, err := yangtree.MarshalJSONIndent(enode, "", " ", yangtree.RepresentItself{})
		if err != nil {
			log.Fatalf("restconf: fault in error report: %v", err)
		}
		return string(b)
	}
	return "unspecified error"
}

func (re *RespError) Response(c *fiber.Ctx) error {
	var marshal func(node yangtree.DataNode, prefix, indent string, option ...yangtree.Option) ([]byte, error)

	c.Set("Server", "open-restconf")
	c.Set("Cache-Control", "no-cache")

	marshal = yangtree.MarshalXMLIndent
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
	}

	if len(re.Errors) > 0 {
		errorsSchema := re.Errors[0].Schema().Parent
		enode, err := yangtree.New(errorsSchema)
		if err != nil {
			log.Fatalf("restconf: errors/error schema not loaded")
		}
		for i := range re.Errors {
			if _, err := enode.Insert(re.Errors[i], nil); err != nil {
				log.Fatalf("restconf: fault in error report: %v", err)
			}
		}
		b, err := marshal(enode, "", " ", yangtree.RepresentItself{})
		if err != nil {
			log.Fatalf("restconf: fault in error report: %v", err)
		}
		return c.Status(re.Code).Send(b)
	}
	c.Status(re.Code)
	return nil
}

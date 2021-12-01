package main

import (
	"log"

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
	ETypeApplication ErrorType = iota
	ETypeProtocol
	ETypeRPC
	ETypeTransport
)

func (et ErrorType) Error() string {
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

func (et ErrorTag) Error() string {
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

func (rctrl *RestconfCtrl) SetError(c *fiber.Ctx, status int, etyp ErrorType, etag ErrorTag, emsg error) error {
	if errorSchema == nil {
		log.Fatalf("restconf: errors schema not loaded")
	}
	e, err := yangtree.NewWithValue(errorSchema.GetSchema("error"),
		map[interface{}]interface{}{
			"error-tag":  etag.Error(),
			"error-type": etyp.Error(),
			"error-path": c.Path(),
		})
	if err != nil {
		log.Fatalf("restconf: errors/error schema not loaded")
	}
	if emsg != nil {
		if fe, ok := emsg.(*fiber.Error); ok {
			if err := yangtree.SetValue(e, "error-message", nil, fe.Message); err != nil {
				log.Fatalf("restconf: fault in error report: %v", err)
			}
		} else {
			if err := yangtree.SetValue(e, "error-message", nil, emsg.Error()); err != nil {
				log.Fatalf("restconf: fault in error report: %v", err)
			}
		}
	}
	rctrl.errors = append(rctrl.errors, e)
	if rctrl.status != fiber.StatusOK {
		rctrl.status = status
	}
	return nil
}

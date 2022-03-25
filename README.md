# Open RESTCONF

This is simple implementation of [RFC8040 RESTCONF protocol](https://datatracker.ietf.org/doc/html/rfc8040) using `gofiber` for the HTTP server facilities of the RESTCONF server and `yangtree` for the YANG-modeled data management.

## RFC8040  RESTCONF Protocol

`RESTCONF` is an HTTP-based protocol that provides a programmatic interface for accessing data defined in YANG([YANG1.0](https://datatracker.ietf.org/doc/html/rfc6020) or [YANG1.1](https://datatracker.ietf.org/doc/html/rfc7950)), using the datastore concepts defined in the Network Configuration Protocol ([NETCONF](https://datatracker.ietf.org/doc/html/rfc6241)).

The `RESTCONF` uses HTTP methods to provide `CRUD` operations on a conceptual datastore containing YANG-defined data, which is compatible with a server that implements NETCONF datastores.

Configuration data and state data are exposed as resources that can be retrieved with the `GET` method. Resources representing configuration data can be modified with the `DELETE`, `PATCH`, `POST`, and `PUT` methods. Data is encoded with either `XML [W3C.REC-xml-20081126]` or `JSON [RFC7159]`.

Data-model-specific RPC operations defined with the YANG `rpc` or `action` statements can be invoked with the `POST` method. Data-model-specific event notifications defined with the YANG `notification` statement can be accessed.

## Open RESTCONF

Open RESTCONF is an open source implementation for the RESTCONF protocol. This application will provide the following features defined in the standard.

> Under development if unmasked!

- [X] Running Datastore managed by the simple API provided by `yangtree`
- [X] Data encoding: `XML`, `JSON`, `YAML`
- [ ] HTTP methods to provide `CRUD` operation for the managed datastore
  - [X] `GET` for the retrieval of the YANG-modeled data
  - [ ] `POST` method for the user-defined YANG `rpc` execution.
  - [ ] `POST` method for the user-defined YANG `action` execution
  - [ ] `POST` method for `edit-config` (nc:operation="create")
  - [ ] `PUT` method for `edit-config` (nc:operation="create/replace)
  - [ ] `PATCH` method for `edit-config` (nc:operation depends on PATCH content)
  - [ ] `DELETE` method for `edit-config` (nc:operation="delete")
- [X] Runtime loading for dynamic datastore schema
- [ ] Datastore management
  - [ ] On-demand callback for YANG-modeled data update
  - [ ] Periodical timer callback for YANG-modeled data update
  - [ ] User-defined RPC execution
- [ ] YANG modules Supported
  - [X] ietf-restconf@2017-01-26 (loaded)
    - [ ] module-state/module/schema (URI) to YANG schema files
  - [X] ietf-yang-library@2016-06-21
    - [ ] must support schema leaf for yang file location
  - [ ] RFC7952 YANG Metadata
- [X] Encoding
  - [X] JSON
  - [X] XML
  - [X] YAML
  - [X] JSON_IETF (RFC7951 - JSON Encoding of Data Modeled with YANG)
- [X] Root Resource Discovery - The client can discover the root of the RESTCONF API by getting the "/.well-known/host-meta" resource and using the `<Link>` element containing the "restconf".

- [ ] 3.5.  Data Resource
  - [ ] 3.5.1.  Timestamp (optional)
  - [ ] 3.5.2.  Entity-Tag (Mandatory)
    - [ ] `ETag`: The server must maintain a resource entity-tag for each resource.
    - [ ] `If-Match` and `If-None-Match`: The server must process `GET` or `HEAD` requests tagged with the conditional header fields and returns one of HTTP Status Code(`202`, `304`) properly.
  - [ ] 3.5.3.  Encoding Data Resource Identifiers in the Request URI
    - [ ] In RESTCONF, URI-encoded path expressions are used instead of XPath Expression.
    - [ ] The server must follow the rule defined in ABNF for RESTCONF Data Resource Identifiers
    - [ ] Leaf-list path format must be supported. (e.g., /restconf/data/top-leaflist=fred).
    - [ ] non-configuration leaf-list exact matching not provided.
    - [ ] Any reserved characters MUST be percent-encoded, according to Sections 2.1 and 2.5 of [RFC3986]. The comma (",") character MUST be percent-encoded if it is present in the key value. e.g. If a first key value is `(,'":" /)`, then the Resource Identifier of the data node becomes `/restconf/data/example-top:top/list1=%2C%27"%3A"%20%2F,,foo`.
    - [ ] A zero-length key value is allowed. e.g. list1=foo,,baz
    - [ ] Note that non-configuration lists are not required to define keys. In this case, a single list instance cannot be accessed.
- [ ] ABNF for Data Resource Identifier: Open RESTCONF provides the Data Resource Identifier defined in RFC 8040 ABNF.
### Media Types (Content-Type) supported

Open RESTCONF provides the following RESTCONF-standard encoding for the YANG-defined data. 

- "application/yang-data+xml"
- "application/yang-data+json"

It also provides other encoding formats lised below.

- "application/yang-data+yaml"
- "text/json", "text/yaml", "text/xml"
- "application/xml", "application/json", "application/yaml"

## RESTCONF Methods

This is HTTP methods that the open-restconf should support.

|RESTCONF  |NETCONF  |
|---------|---------|
|OPTIONS |none                                                     |
|HEAD    | `<get-config>`, `<get>`                                 |
|GET     | `<get-config>`, `<get>`                                 |
|POST    | `<edit-config>` (nc:operation="create")                 |
|POST    | invoke an RPC operation                                 |
|PUT     | `<copy-config>` (PUT on rctrl)                          |
|PUT     | `<edit-config>` (nc:operation="create/replace")         |
|PATCH   | `<edit-config>` (nc:operation depends on PATCH content) |
|DELETE  | `<edit-config>` (nc:operation="delete")                 |


### OPTIONS method

OPTIONS is used to check the PATCH method is available.

```txt
   The "Accept-Patch" header field MUST be supported and returned in the
   response to the OPTIONS request, as defined in [RFC5789].
```

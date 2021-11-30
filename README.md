# RESTCONF

`RESTCONF` is a network management protocol based on HTTP [RFC7230], for configuring data defined in `YANG version 1 [RFC6020]` or `YANG version 1.1 [RFC7950]`, using the datastore concepts defined in `NETCONF [RFC6241]`.

`RESTCONF` uses HTTP methods to provide `CRUD` operations on a conceptual datastore containing YANG-defined data, which is compatible with a server that implements NETCONF datastores.

Configuration data and state data are exposed as resources that can be retrieved with the `GET` method. Resources representing configuration data can be modified with the `DELETE`, `PATCH`, `POST`, and `PUT` methods. Data is encoded with either `XML [W3C.REC-xml-20081126]` or `JSON [RFC7159]`.

Data-model-specific RPC operations defined with the YANG `rpc` or `action` statements can be invoked with the `POST` method. Data-model-specific event notifications defined with the YANG `notification` statement can be accessed.

> https://datatracker.ietf.org/doc/html/rfc8040


```bash
wget -q -O - http://localhost:10000/articles
wget -q -O - http://localhost:10000/articles/1
wget -q -O - http://localhost:10000/articles/2
wget --post-file test.json -q -O - http://localhost:10000/articles
```

## RESTCONF Media Types (Content-Type)

- "application/yang-data+xml"
- "application/yang-data+json"

### "application/yang-data+json"

- Each conceptual YANG data node is encoded according to [RFC7951].
- A metadata annotation is encoded according to [RFC7952].

## common RESTful operations

| HTTP Verb  | CRUD   | Entire Collection (e.g. /customers)  | Specific Item (e.g. /customers/{id})   |
|------------|--------------|---|---|
|POST        |Create        |201 (Created), 'Location' header with link to /customers/{id} containing new ID.|404 (Not Found), 409 (Conflict) if resource already exists..|
|GET         |Read          |200 (OK), list of customers. Use pagination, sorting and filtering to navigate big lists.|200 (OK), single customer. 404 (Not Found), if ID not found or invalid.|
|PUT         |Update/Replace|405 (Method Not Allowed), unless you want to update/replace every resource in the entire collection.|200 (OK) or 204 (No Content). 404 (Not Found), if ID not found or invalid.|
|PATCH       |Update/Modify |405 (Method Not Allowed), unless you want to modify the collection itself.|200 (OK) or 204 (No Content). 404 (Not Found), if ID not found or invalid.|
|DELETE      |Delete        |405 (Method Not Allowed), unless you want to delete the whole collection—not often desirable.|200 (OK). 404 (Not Found), if ID not found or invalid.|

## RESTCONF Methods

```text
   +----------+-------------------------------------------------------+
   | RESTCONF | NETCONF                                               |
   +----------+-------------------------------------------------------+
   | OPTIONS  | none                                                  |
   |          |                                                       |
   | HEAD     | <get-config>, <get>                                   |
   |          |                                                       |
   | GET      | <get-config>, <get>                                   |
   |          |                                                       |
   | POST     | <edit-config> (nc:operation="create")                 |
   |          |                                                       |
   | POST     | invoke an RPC operation                               |
   |          |                                                       |
   | PUT      | <copy-config> (PUT on datastore)                      |
   |          |                                                       |
   | PUT      | <edit-config> (nc:operation="create/replace")         |
   |          |                                                       |
   | PATCH    | <edit-config> (nc:operation depends on PATCH content) |
   |          |                                                       |
   | DELETE   | <edit-config> (nc:operation="delete")                 |
   +----------+-------------------------------------------------------+
```

### OPTIONS method

OPTIONS is used to check the PATCH method is available.

```txt
   The "Accept-Patch" header field MUST be supported and returned in the
   response to the OPTIONS request, as defined in [RFC5789].
```


## YANG library

RESTCONF utilizes `YANG library [RFC7895]` to allow a client to discover the YANG module conformance information for the server, in case the client wants to use it.

- 지원 model의 listing, discover 및 download 지원

## YANG Modules for RESTCONF

- ietf-restconf-monitoring
- ietf-yang-library

## Transport Protocol

- Hypertext Transfer Protocol (HTTP/1.1): Message Syntax and Routing [RFC7230]
  - Given the nearly ubiquitous support for HTTP over TLS [RFC7230], RESTCONF implementations MUST support the "https" URI scheme, which has the IANA-assigned default port 443.
  - RESTCONF servers MUST present an X.509v3-based certificate when establishing a TLS connection with a RESTCONF client.  The use of X.509v3-based certificates is consistent with NETCONF over TLS [RFC7589].
- Transport Layer Security (TLS) protocol [RFC5246]
- Recommendations for Secure Use of Transport Layer Security (TLS) and Datagram Transport Layer Security (DTLS) [RFC7525]
- The RESTCONF client MUST either
  - (1) use X.509 certificate path validation [RFC5280] to verify the integrity of the RESTCONF server's TLS certificate or
  - (2) match the server's TLS certificate with a certificate obtained by a trusted mechanism (e.g., a pinned certificate).
  - If the above two conditions are failed, the connection MUST be terminated, as described in Section 7.2.1 of [RFC5246].
- The RESTCONF client MUST check the identity of the server according to Section 3.1 of [RFC2818].
- Authenticated Client Identity
  - Section 3.1 of [RFC7235] -  Hypertext Transfer Protocol (HTTP/1.1): Authentication > 401 Unauthorized
  - Section 7.4.6 of [RFC5246] - The Transport Layer Security (TLS) Protocol Version 1.2 > Client Certificate
  - Section 5.1 in [RFC7235] - Hypertext Transfer Protocol (HTTP/1.1): Authentication > Authentication Scheme Registry
  - NETCONF Access Control Model (NACM) [RFC6536] - Network Configuration Protocol (NETCONF) Access Control Model
  - Section 7 of [RFC7589] - Using the NETCONF Protocol over Transport Layer Security (TLS) with Mutual X.509 Authentication > Client Identity

## Requirements

- [ ] XML and JSON serialization support
  - [ ] application/yang-data+xml
  - [ ] application/yang-data+json
- [ ] Root Resource Discovery - The client can discover the root of the RESTCONF API by getting the "/.well-known/host-meta" resource ([RFC6415]) and using the `<Link>` element containing the "restconf" attribute.
- [ ] API Resource
  - [ ] {+restconf}/data
  - [ ] {+restconf}/operations
  - [ ] {+restconf}/yang-library-version


## RESTCONF URI ROUTES

- `/.well-known/host-meta`


## CURL or httpie example

```bash
curl -v --header "Accept: application/xrd+xml" localhost:3000/.well-known/host-meta 2> >(sed '/^*/d')
curl --header "Accept: application/xrd+xml" localhost:3000/.well-known/host-meta
```

## Request and Response

```http
      GET /restconf/data/example-jukebox:jukebox/library\
          ?content=nonconfig HTTP/1.1
      Host: example.com
      Accept: application/yang-data+xml


      HTTP/1.1 200 OK
      Date: Thu, 26 Jan 2017 20:56:30 GMT
      Server: example-server
      Cache-Control: no-cache
      Content-Type: application/yang-data+xml

      <library xmlns="https://example.com/ns/example-jukebox">
        <artist-count>42</artist-count>
        <album-count>59</album-count>
        <song-count>374</song-count>
      </library>
```

### Response Header

- Server: open-restconf
- Cache-Control: no-cache

## Requirements

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

### ABNF for Data Resource Identifier

```txt
   api-path = root *("/" (api-identifier / list-instance))

   root = string  ;; replacement string for {+restconf}

   api-identifier = [module-name ":"] identifier

   module-name = identifier

   list-instance = api-identifier "=" key-value *("," key-value)

   key-value = string  ;; constrained chars are percent-encoded

   string = <an unquoted string>

   identifier = (ALPHA / "_")
                *(ALPHA / DIGIT / "_" / "-" / ".")
```
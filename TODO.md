## common RESTful operations

| HTTP Verb  | CRUD   | Entire Collection (e.g. /customers)  | Specific Item (e.g. /customers/{id})   |
|------------|--------------|---|---|
|POST        |Create        |201 (Created), 'Location' header with link to /customers/{id} containing new ID.|404 (Not Found), 409 (Conflict) if resource already exists..|
|GET         |Read          |200 (OK), list of customers. Use pagination, sorting and filtering to navigate big lists.|200 (OK), single customer. 404 (Not Found), if ID not found or invalid.|
|PUT         |Update/Replace|405 (Method Not Allowed), unless you want to update/replace every resource in the entire collection.|200 (OK) or 204 (No Content). 404 (Not Found), if ID not found or invalid.|
|PATCH       |Update/Modify |405 (Method Not Allowed), unless you want to modify the collection itself.|200 (OK) or 204 (No Content). 404 (Not Found), if ID not found or invalid.|
|DELETE      |Delete        |405 (Method Not Allowed), unless you want to delete the whole collection—not often desirable.|200 (OK). 404 (Not Found), if ID not found or invalid.|

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

      HTTP/1.1 400 Bad Request
      Date: Thu, 26 Jan 2017 20:56:30 GMT
      Server: example-server
      Content-Type: application/yang-data+json

      { "ietf-restconf:errors" : {
          "error" : [
            {
              "error-type" : "protocol",
              "error-tag" : "invalid-value",
              "error-path" : "/example-ops:input/delay",
              "error-message" : "Invalid input parameter"
            }
          ]
        }
      }
```

### ABNF for Data Resource Identifier

Open RESTCONF provides the Data Resource Identifier defined in RFC 8040 ABNF.

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

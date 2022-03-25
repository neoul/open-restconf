#!/bin/bash

set -v

http POST http://127.0.0.1:8080/restconf/operations/reboot Accept:application/yang-data+json input:='{"delay":10, "message": "reboot message", "language":"en-US"}'
http POST http://127.0.0.1:8080/restconf/operations/get-reboot-info Accept:application/yang-data+json input:='{"delay":10, "message": "reboot message", "language":"en-US"}'

http --body GET http://localhost:8080/restconf/yang-library-version Accept:text/txt

http --body GET http://localhost:8080/restconf/yang-library-version Accept:application/yang-data+xml
http --body GET http://localhost:8080/restconf/yang-library-version Accept:application/yang-data+json
http --body GET http://localhost:8080/restconf/yang-library-version Accept:application/yang-data+yaml

http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/yang-data+xml
http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/yang-data+json
http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/yang-data+yaml

http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/yang-data+xml
http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/yang-data+json
http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/yang-data+yaml

http GET http://localhost:8080/restconf/data/modules-state/module Accept:application/yang-data+xml
http GET http://localhost:8080/restconf/data/modules-state/module Accept:application/yang-data+json
http GET http://localhost:8080/restconf/data/modules-state/module Accept:application/yang-data+yaml

# http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+xml
# http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+json
# http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+yaml

# http POST http://127.0.0.1:8080/restconf/operations/reboot Accept:application/yang-data+json input:='{"delay":10, "message": "reboot message", "language":"KR"}'


# curl -v --header "Accept: application/yang-data+xml" -H "Content-Type: application/json" -X POST http://localhost:8080/restconf/operations/reboot -d '{"input":{"message": "superman", "delay" : 30, "language":"english"}}'

# wget -q -O - http://127.0.0.1:8080/.well-known/host-meta -d
# wget -d -q -O - http://127.0.0.1:8080/restconf/data/jukebox/playlist=Foo-One/song=2
# wget --post-file test.json -q -O - http://localhost:10000/articles

# curl -v --header "Accept: application/xrd+xml" localhost:3000/.well-known/host-meta 2> >(sed '/^*/d')
# curl --header "Accept: application/xrd+xml" localhost:3000/.well-known/host-meta
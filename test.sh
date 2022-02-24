#!/bin/bash

set -e

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

# http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+xml
# http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+json
# http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+yaml

# http POST http://127.0.0.1:8080/restconf/operations/reboot Accept:application/yang-data+json input:='{"delay":10, "message": "reboot message", "language":"KR"}'
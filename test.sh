#!/bin/bash

RUN() {
    echo $*
    $*
}

http --body GET http://localhost:8080/restconf/yang-library-version Accept:text/txt

RUN http --body GET http://localhost:8080/restconf/yang-library-version Accept:application/yang-data+xml
RUN http --body GET http://localhost:8080/restconf/yang-library-version Accept:application/yang-data+json
RUN http --body GET http://localhost:8080/restconf/yang-library-version Accept:application/yang-data+yaml

RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/yang-data+xml
RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/yang-data+json
RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/yang-data+yaml

RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/yang-data+xml
RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/yang-data+json
RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/yang-data+yaml

# RUN http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+xml
# RUN http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+json
# RUN http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yang-data+yaml

RUN http POST http://127.0.0.1:8080/restconf/operations/reboot Accept:application/yang-data+json input:='{"delay":10, "message": "reboot message", "language":"KR"}'
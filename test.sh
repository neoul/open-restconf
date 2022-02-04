#!/bin/bash

RUN() {
    echo $*
    $*
}

RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/xml
RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/json
RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library/artist/name Accept:application/yaml

# RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/xml
# RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/json
# RUN http --body GET http://127.0.0.1:8080/restconf/data/jukebox/library Accept:application/yaml

# RUN http --body GET http://127.0.0.1:8080/restconf/data Accept:application/xml
# RUN http --body GET http://127.0.0.1:8080/restconf/data Accept:application/json
# RUN http --body GET http://127.0.0.1:8080/restconf/data Accept:application/yaml
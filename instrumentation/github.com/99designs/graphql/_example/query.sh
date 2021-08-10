#!/usr/bin/env sh
wget  "http://${1}:8080/query" \
--header 'Accept-Encoding: gzip, deflate, br' \
--header 'Content-Type: application/json' \
--header 'Accept: application/json' \
--header 'Connection: keep-alive'  \
--header 'DNT: 1' \
--post-data '{"query":"query ping {\n  ping{\n    id\n  }\n}"}'
#!/bin/bash

# Copyright The OpenTelemetry Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script generates 3 sets of ECSDA (prime256v1 curve) certificates and keys. The
# expiration date is set to 1000000 days from creation.

# Generate a certificate authority certificate and key.
openssl ecparam -genkey -name prime256v1 -out ca.key
openssl req -nodes -x509 -days 1000000 -new -SHA256 -key ca.key -out ca.crt \
    -subj "/C=US/ST=State/L=State/O=Org/CN=CommonName"

# Generate a certificate and key for the server that is signed by the CA/
openssl ecparam -genkey -name prime256v1 -out server.key
openssl req -nodes -new -SHA256 -key server.key -out server.csr \
    -subj "/C=US/ST=State/L=State/O=Org/CN=CommonName"
openssl x509 -req -days 1000000 -SHA256 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt -extfile <(printf "subjectAltName = IP:127.0.0.1")

# Generate a certificate and key for the client that is signed by the CA.
openssl ecparam -genkey -name prime256v1 -out client.key
openssl req -nodes -new -SHA256 -key client.key -out client.csr \
    -subj "/C=US/ST=State/L=State/O=Org/CN=CommonName"
openssl x509 -req -days 1000000 -SHA256 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

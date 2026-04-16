#!/bin/bash
set -e

# Configuration
DAYS_VALID=3650
SUBJ_CA="/C=US/ST=California/L=San Francisco/O=My Company/CN=My CA"
SUBJ_SERVER="/C=US/ST=California/L=San Francisco/O=My Company/CN=localhost"
SUBJ_CLIENT="/C=US/ST=California/L=San Francisco/O=My Company/CN=client"

# Create CA
openssl genrsa -out ca.key 2048
openssl req -x509 -new -nodes -key ca.key -sha256 -days $DAYS_VALID -out ca.crt -subj "$SUBJ_CA"

# Create Server Cert
mkdir -p server-certs
openssl genrsa -out server-certs/server.key 2048
openssl req -new -key server-certs/server.key -out server-certs/server.csr -subj "$SUBJ_SERVER"
openssl x509 -req -in server-certs/server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server-certs/server.crt -days $DAYS_VALID -sha256 -extfile <(echo "subjectAltName=DNS:localhost,IP:127.0.0.1")

# Create Client Cert
mkdir -p client-certs
openssl genrsa -out client-certs/client.key 2048
openssl req -new -key client-certs/client.key -out client-certs/client.csr -subj "$SUBJ_CLIENT"
openssl x509 -req -in client-certs/client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client-certs/client.crt -days $DAYS_VALID -sha256

# Copy to other expected locations
cp client-certs/client.crt client.crt
cp client-certs/client.key client.key

# Clean up temporary files but KEEP .csr as they were tracked in repo
rm ca.srl ca.key

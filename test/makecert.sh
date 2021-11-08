#!/bin/bash

# # if you need a new self-signed cert to test https
openssl genrsa -out "tls.key" 4096
openssl req -new -key "tls.key" -out "tls.csr" -subj '/C=FI/ST=Uusimaa/L=Espoo/O=CSC/CN=localhost'
openssl x509 -req -days 365 -in "tls.csr" -signkey "tls.key" -out "tls.crt"

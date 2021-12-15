#!/bin/bash

# create certs
source ./makecert.sh

# https://github.com/elixir-oslo/crypt4gh
curl -fsSL https://raw.githubusercontent.com/elixir-oslo/crypt4gh/master/install.sh | sh -s -- -b .

# run crypt4gh to generate password
./crypt4gh generate --name=download --password=passphrase
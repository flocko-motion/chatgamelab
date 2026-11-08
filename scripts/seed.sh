#!/bin/bash

# seed testdata into whatever server the user is currently logged into
# so make sure to first:
# - cd to go
# - go run . user login (user --help to learn about options)
# - now run this script

set -ex

cd "$(dirname "$0")/../go"

time go run . game put ../testdata/games

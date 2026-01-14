#!/bin/bash

set -ex

cd "$(dirname "$0")/server"

time go run . lang translate --model ministral-8b-latest --input ../web/src/i18n/locales --output ../web/src/i18n/locales

go run . lang list --json > ../web/src/i18n/locales/languages.json

rm ./lang/locales/*.json
cp ../web/src/i18n/locales/*.json ./lang/locales/.
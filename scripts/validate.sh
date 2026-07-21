#!/usr/bin/env sh
set -eu

find src public resources/views tests -name '*.php' -print0 | xargs -0 -n1 php -l >/dev/null
node --check public/assets/app.js >/dev/null
php tests/run.php

if find . -type f \( -name '*.go' -o -name 'go.mod' -o -name 'go.sum' \) | grep -q .; then
  echo 'Ditemukan file/runtime Go dalam paket PHP.' >&2
  exit 1
fi

echo 'Validasi PHP-only selesai.'

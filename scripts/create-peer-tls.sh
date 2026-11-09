#!/usr/bin/env bash
# Bootstrap a dedicated CA. For enterprise PKI, supply equivalent certificates.
set -euo pipefail
out=${1:-.local/peer-tls}
if [[ -e "$out/ca.key" || -e "$out/tls.key" ]]; then
  echo 'Refusing to overwrite existing keys; choose a new directory for rotation.' >&2
  exit 1
fi
umask 077
mkdir -p "$out"
openssl req -x509 -newkey rsa:3072 -sha256 -nodes -days 3650 \
  -subj '/CN=ADHD Duck dedicated CA' -keyout "$out/ca.key" -out "$out/ca.crt" \
  -addext 'basicConstraints=critical,CA:TRUE' -addext 'keyUsage=critical,keyCertSign,cRLSign'
openssl req -new -newkey rsa:3072 -sha256 -nodes \
  -subj '/CN=adhdduck.internal' -keyout "$out/tls.key" -out "$out/tls.csr"
cat > "$out/extensions.cnf" <<'EXT'
basicConstraints=critical,CA:FALSE
keyUsage=critical,digitalSignature,keyEncipherment
extendedKeyUsage=serverAuth,clientAuth
subjectAltName=DNS:adhdduck.internal
EXT
openssl x509 -req -sha256 -days 90 -in "$out/tls.csr" -CA "$out/ca.crt" \
  -CAkey "$out/ca.key" -CAcreateserial -extfile "$out/extensions.cnf" -out "$out/tls.crt"
openssl verify -CAfile "$out/ca.crt" -purpose sslserver "$out/tls.crt"
openssl verify -CAfile "$out/ca.crt" -purpose sslclient "$out/tls.crt"
printf 'Created certificates in %s. Keep ca.key offline; never upload it to Kubernetes.\n' "$out"

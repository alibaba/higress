RSA_KEY_LENGTH="${RSA_KEY_LENGTH:-4096}"

VOLUMES_ROOT="helm/core/files"

initializeApiServer() {
  echo "Initializing API server configurations..."

  mkdir -p "$VOLUMES_ROOT/api" && cd "$_"
  checkExitCode "Creating volume for API server fails with $?"

  if [ ! -f ca.key ] || [ ! -f ca.crt ]; then
    echo "  Generating CA certificate...";
    openssl req -nodes -new -x509 -days 36500 -keyout ca.key -out ca.crt -subj "/CN=higress-root-ca/O=higress" > /dev/null 2>&1
    checkExitCode "  Generating CA certificate for API server fails with $?";
  else
    echo "  CA certificate already exists.";
  fi
  if [ ! -f server.key ] || [ ! -f server.crt ]; then
    echo "  Generating server certificate..."
    openssl req -out server.csr -new -newkey rsa:$RSA_KEY_LENGTH -nodes -keyout server.key -subj "/CN=higress-api-server/O=higress" > /dev/null 2>&1 \
      && openssl x509 -req -days 36500 -in server.csr -CA ca.crt -CAkey ca.key -set_serial 01 -sha256 -out server.crt > /dev/null 2>&1
    checkExitCode "  Generating server certificate fails with $?";
  else
    echo "  Server certificate already exists.";
  fi
  if [ ! -f nacos.key ]; then
    echo "  Generating data encryption key..."
    if [ -z "$NACOS_DATA_ENC_KEY" ]; then
      cat /dev/urandom | tr -dc '[:graph:]' | head -c 32 > nacos.key
    else
      echo -n "$NACOS_DATA_ENC_KEY" > nacos.key
    fi
  else
    echo "  Client certificate already exists.";
  fi
  if [ ! -f client.key ] || [ ! -f client.crt ]; then
    echo "  Generating client certificate..."
    openssl req -out client.csr -new -newkey rsa:$RSA_KEY_LENGTH -nodes -keyout client.key -subj "/CN=higress/O=system:masters" > /dev/null 2>&1 \
      && openssl x509 -req -days 36500 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 02 -sha256 -out client.crt > /dev/null 2>&1
    checkExitCode "  Generating client certificate fails with $?";
  else
    echo "  Client certificate already exists.";
  fi
  echo "  Done."
}

checkExitCode() {
  # $1 message
  retVal=$?
  if [ $retVal -ne 0 ]; then
    echo ${1:-"  Command fails with $retVal"}
    exit $retVal
  fi
}

initializeApiServer
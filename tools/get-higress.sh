#!/usr/bin/env bash

#  Copyright (c) 2023 Alibaba Group Holding Ltd.

#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at

#       http:www.apache.org/licenses/LICENSE-2.0

#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

export VERSION

MODE="install"

HAS_CURL="$(type "curl" &> /dev/null && echo true || echo false)"
HAS_WGET="$(type "wget" &> /dev/null && echo true || echo false)"
HAS_DOCKER="$(type "docker" &> /dev/null && echo true || echo false)"

parseArgs() {
  CONFIG_ARGS=()

  DESTINATION=""
  MODE="install"

  if [[ $1 != "-"* ]]; then
    DESTINATION="$1"
    shift
  fi

  while [[ $# -gt 0 ]]; do
    case $1 in
      -h|--help)
        outputUsage
        exit 0
        ;;
      -u|--update)
        MODE="update"
        shift
        ;;
      *)
        CONFIG_ARGS+=("$1")
        shift
        ;;
    esac
  done

  DESTINATION=${DESTINATION:-$PWD/higress}
}

validateArgs() {
  if [ -d "$DESTINATION" ]; then
    if [ -e "${DESTINATION}/compose/.configured" -a "$MODE" != "update" ]; then
      echo "Higress is already installed in the target folder \"$DESTINATION\". Add \"-u\" to update an existed Higress instance." && exit 1
    fi
    if [ ! -w "$DESTINATION" ]; then
      echo "The target folder \"$DESTINATION\" is not writeable." && exit 1
    fi
  else
    if [ "$MODE" == "update" ]; then
      echo "The target folder \"$DESTINATION\" for update doesn't exist." && exit 1
    fi
    mkdir -p "$DESTINATION"
    if [ $? -ne 0 ]; then
      exit 1
    fi
  fi

  cd "$DESTINATION"
  DESTINATION=$(pwd -P)
  cd - > /dev/null
}

outputUsage() {
  echo "Usage: $(basename -- "$0") [DIR] [OPTIONS...]"
  echo 'Install Higress (standalone version) into the DIR ("./higress" by default).'
  echo '
 -c, --config-url=URL       URL of the config storage
                            Use Nacos with format: nacos://192.168.0.1:8848
                            Use local files with format: file:///opt/higress/conf
     --use-builtin-nacos    use the built-in Nacos service instead of
                            an external one
     --nacos-ns=NACOS-NAMESPACE
                            the ID of Nacos namespace to store configurations
                            default to "higress-system" if unspecified
     --nacos-username=NACOS-USERNAME
                            the username used to access Nacos
                            only needed if auth is enabled in Nacos
     --nacos-password=NACOS-PASSWORD
                            the password used to access Nacos
                            only needed if auth is enabled in Nacos
 -k, --data-enc-key=KEY     the key used to encrypt sensitive configurations
                            MUST contain 32 characters
                            A random key will be generated if unspecified
 -p, --console-password=CONSOLE-PASSWORD
                            the password to be used to visit Higress Console
                            default to "admin" if unspecified
     --nacos-port=NACOS-PORT
                            the HTTP port used to access the built-in Nacos
                            default to 8848 if unspecified
     --gateway-http-port=GATEWAY-HTTP-PORT
                            the HTTP port to be listened by the gateway
                            default to 80 if unspecified
     --gateway-https-port=GATEWAY-HTTPS-PORT
                            the HTTPS port to be listened by the gateway
                            default to 443 if unspecified
     --gateway-metrics-port=GATEWAY-METRICS-PORT
                            the metrics port to be listened by the gateway
                            default to 15020 if unspecified
     --console-port=CONSOLE-PORT
                            the port used to visit Higress Console
                            default to 8080 if unspecified
 -u, --update               update an existed Higress instance.
                            no user configuration will be changed during update.
 -h, --help                 give this help list'
}

# initArch discovers the architecture for this system.
initArch() {
  ARCH=$(uname -m)
  case $ARCH in
    armv5*) ARCH="armv5";;
    armv6*) ARCH="armv6";;
    armv7*) ARCH="arm";;
    aarch64) ARCH="arm64";;
    x86) ARCH="386";;
    x86_64) ARCH="amd64";;
    i686) ARCH="386";;
    i386) ARCH="386";;
  esac
}

# initOS discovers the operating system for this system.
initOS() {
  OS="$(uname|tr '[:upper:]' '[:lower:]')"
  case "$OS" in
    # Minimalist GNU for Windows
    mingw*|cygwin*) OS='windows';;
  esac
}

# runs the given command as root (detects if we are root already)
runAsRoot() {
  if [ $EUID -ne 0 ]; then
    sudo "${@}"
  else
    "${@}"
  fi
}

# verifySupported checks that the os/arch combination is supported for
# binary builds, as well whether or not necessary tools are present.
verifySupported() {
  local supported="darwin-amd64\nlinux-amd64\nwindows-amd64\ndarwin-arm64\nlinux-arm64\nwindows-arm64\n"
  if ! echo "${supported}" | grep -q "${OS}-${ARCH}"; then
    echo "${OS}-${ARCH} platform isn't supported at the moment."
    echo "Stay tuned for updates on https://github.com/alibaba/higress."
    exit 1
  fi

  if [ "${HAS_CURL}" != "true" ] && [ "${HAS_WGET}" != "true" ]; then
    echo "Either curl or wget is required"
    exit 1
  fi

  if [ "${HAS_DOCKER}" != "true" ]; then
    echo "Docker is required"
    exit 1
  fi
}

REPO_BASE_URL="https://higress.io/standalone"

# checkDesiredVersion checks if the desired version is available.
checkDesiredVersion() {
  if [ -z "$VERSION" ]; then
    local version_url="${REPO_BASE_URL}/VERSION"
    if [ "${HAS_CURL}" == "true" ]; then
      VERSION=$(curl -Ls $version_url)
    elif [ "${HAS_WGET}" == "true" ]; then
      VERSION=$(wget $version_url -O - 2>/dev/null)
    fi
  fi
}

# download downloads the latest package
download() {
  HIGRESS_DIST="${VERSION}.tar.gz"
  DOWNLOAD_URL="${REPO_BASE_URL}/higress-${VERSION}.tar.gz"
  HIGRESS_TMP_ROOT="$(mktemp -dt higress-installer-XXXXXX)"
  HIGRESS_TMP_FILE="$HIGRESS_TMP_ROOT/$HIGRESS_DIST"
  echo "Downloading $DOWNLOAD_URL..."
  if [ "${HAS_CURL}" == "true" ]; then
    curl -SsL "$DOWNLOAD_URL" > "$HIGRESS_TMP_FILE"
  elif [ "${HAS_WGET}" == "true" ]; then
    wget -q -O - "$DOWNLOAD_URL" > "$HIGRESS_TMP_FILE"
  fi
}

# install installs the product.
install() {
  tar -zx --exclude="docs" --exclude="src" --exclude="test" -f "$HIGRESS_TMP_FILE" -C "$DESTINATION" --strip-components=1
  echo -n "$VERSION" > "$DESTINATION/VERSION"
  bash "$DESTINATION/bin/configure.sh" --auto-start ${CONFIG_ARGS[@]}
}

# update updates the product.
update() {
  CURRENT_VERSION="0.0.0"
  if [ -f "$DESTINATION/VERSION" ]; then
    CURRENT_VERSION="$(cat "$DESTINATION/VERSION")"
  fi
  if [ "$CURRENT_VERSION" == "$VERSION" ]; then
    echo "Higress is already up-to-date."
    exit 0
  fi

  BACKUP_FOLDER="$(cd ${DESTINATION}/.. ; pwd)"
  BACKUP_FILE="${BACKUP_FOLDER}/higress_backup_$(date '+%Y%m%d%H%M%S').tar.gz" 
  tar -zc -f "$BACKUP_FILE" -C "$DESTINATION" .
  echo "The current version is packed here: $BACKUP_FILE"
  echo ""

  download
  echo ""

  tar -zx --exclude="docs" --exclude="src" --exclude="test" --exclude="compose/.env" -f "$HIGRESS_TMP_FILE" -C "$DESTINATION" --strip-components=1
  tar -zx -f "$HIGRESS_TMP_FILE" -C "$DESTINATION" --transform='s/env/env_new/g' --strip-components=1 "higress-standalone-${VERSION#v}/compose/.env"
  bash "$DESTINATION/bin/update.sh"
  echo -n "$VERSION" > "$DESTINATION/VERSION"
  return
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [ -n "$INPUT_ARGUMENTS" ]; then
      echo "Failed to ${MODE} Higress with the arguments provided: $INPUT_ARGUMENTS"
    else
      echo "Failed to ${MODE} Higress"
    fi
    echo -e "\tFor support, go to https://github.com/alibaba/higress."
  fi
  exit $result
}

# cleanup temporary files.
cleanup() {
  if [[ -d "${HIGRESS_TMP_ROOT:-}" ]]; then
    rm -rf "$HIGRESS_TMP_ROOT"
  fi
}

parseArgs "$@"
validateArgs

# Stop execution on any error
trap "fail_trap" EXIT
set -e

initArch
initOS
verifySupported

checkDesiredVersion
case "$MODE" in
  update)
    update
    ;;
  *)
    download
    install
    ;;
esac
cleanup
#!/bin/bash

INPUTS=("${@}")
TARGET_ARCH=${TARGET_ARCH:-amd64}
DOCKER_WORKING_DIR=${INPUTS[${#INPUTS[@]}-1]}
FILES=("${INPUTS[@]:0:${#INPUTS[@]}-1}")

set -eu;

function may_copy_into_arch_named_sub_dir() {
  FILE=${1}
  COPY_ARCH_RELATED=${COPY_ARCH_RELATED:-1}

  FILE_INFO=$(file "${FILE}" || true)
  # when file is an `ELF 64-bit LSB`,
  # will put an arch named sub dir
  # like
  #   arm64/
  #   amd64/
  if [[ ${FILE_INFO} == *"ELF 64-bit LSB"* ]]; then
    chmod 755 "${FILE}"

    case ${FILE_INFO} in
      *x86-64*)
        mkdir -p "${DOCKER_WORKING_DIR}/amd64/" && cp -rp "${FILE}" "${DOCKER_WORKING_DIR}/amd64/"
        ;;
      *aarch64*)
        mkdir -p "${DOCKER_WORKING_DIR}/arm64/" && cp -rp "${FILE}" "${DOCKER_WORKING_DIR}/arm64/"
        ;;
        *)
        cp -rp "${FILE}" "${DOCKER_WORKING_DIR}"
        ;;
    esac


    if [[ ${COPY_ARCH_RELATED} == 1 ]]; then
      # if other arch files exists, should copy too.
      for ARCH in "amd64" "arm64"; do
        # like file `out/linux_amd64/pilot-discovery`
        # should check  `out/linux_arm64/pilot-discovery` exists then do copy

        FILE_ARCH_RELATED=${FILE/linux_${TARGET_ARCH}/linux_${ARCH}}

        if [[ ${FILE_ARCH_RELATED} != "${FILE}" && -f ${FILE_ARCH_RELATED} ]]; then
          COPY_ARCH_RELATED=0 may_copy_into_arch_named_sub_dir "${FILE_ARCH_RELATED}"
        fi
      done
    fi

  else
    cp -rp "${FILE}" "${DOCKER_WORKING_DIR}"
  fi
}


for FILE in "${FILES[@]}"; do
  may_copy_into_arch_named_sub_dir "${FILE}"
done

ls "${DOCKER_WORKING_DIR}";

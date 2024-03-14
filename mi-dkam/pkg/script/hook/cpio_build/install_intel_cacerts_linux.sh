#!/bin/bash

# Install Intel CA Internal Certs on a Linux Host

errexit() {
   echo "ERROR: $*" 1>&2
   exit 1
}

#======================================================================================================================
# MAIN
#======================================================================================================================
REPO_PATH="https://ubit-artifactory-or.intel.com/artifactory/it-btrm-local/intel_cacerts"

#os_distro="$(lsb_release -i | awk '{print $3'})" || errexit "Failed to find os_distro"
os_distro="$(awk -F '=' '/^ID=/ { gsub(/"/, "", $2); print($2); }' /etc/os-release)" || errexit "Failed to find os_distro"
[ -z "$os_distro" ] && errexit "Empty os_distro"

declare -a cert_file_list=()
for alias_name in intel_5A intel_5A_2 intel_5B intel_5B_2 intel_root
do
  cert_file=${alias_name}.crt
  artifact="${REPO_PATH}/${cert_file}"
  echo "Fetching $artifact"
  curl -LO --insecure -s "$artifact" || errexit "Failed to fetch $artifact"
  [ -f ${cert_file} ] || errexit "Failed to find $cert_file"
  cert_file_list+=($cert_file)
done

echo "Installing certs on $os_distro"
case "$os_distro" in
  ubuntu)
    cert_loc=/usr/local/share/ca-certificates
    update_cmd=update-ca-certificates
    ;;

  rhel)
    cert_loc=/etc/pki/ca-trust/source/anchors
    update_cmd=update-ca-trust
    ;;

  sles)
    cert_loc=/usr/share/pki/trust/anchors
    update_cmd=update-ca-certificates
    ;;

  *)
    errexit "Unsupported distro: $os_distro"
    ;;

esac

[ -d $cert_loc ] || errexit "Cert location $cert_loc is not found or not a directory"
cp ${cert_file_list[@]} $cert_loc || errexit "Failed to copy cert files to $cert_loc"
$update_cmd || errexit "Failed to update certificates"

echo "Successfully installed Intel Internal CA Certs ${cert_file_list[@]}"

#!/bin/bash
set -Eu

Help()
{
    # Display Help
    echo "This script is used to run svi APIs to send TC provision script to device"
    echo
    echo "Syntax: ./$0 [-c|h|o|p|s|t|u|v]"
    echo "options:"
    echo "c     Path to directory where both client and CA certs of owner service are present. Used for mTLS/Client certificate Authentication."
    echo "o     Owner IP, if not provided defaults to host.docker.internal"
    echo "p     Owner service port, if not provided defaults to "
    echo "s     Bash script file name"
    echo "t     Thingsboard IP"
    echo "u     Thingsboard port"
    echo "v     Thingsboard token"
    echo "h     Help."
    echo
}

while getopts "c:h:o:p:s:t:u:v:" flag;
do
    case "${flag}" in
        c) cert_path=${OPTARG};;
        h) Help
           exit 0;;
        o) owner_ip=${OPTARG};;
        p) owner_svc_port=${OPTARG};;
        s) script_file_name=${OPTARG};;
        \?) echo "Error: Invalid Option, use -h for help"
            exit 1;;
    esac
done

if [ ! -d "${cert_path}" ]; then
        echo "Owner service certificates path not exists!!"
        exit 1
fi

if [ ! -f "${script_file_name}" ]; then
        echo "Script file $script_file_name not exists!!"
        exit 1
fi
if [ -z "${owner_ip}" -o -z "${owner_svc_port}" -o -z "${script_file_name}"]; then
	echo "Missing argument and all are mandatory"
	Help
	exit 1
fi

# Use Owner API to give instructions to transfer file (tc_provision_tb.sh) from the Owner service to the device.
response=$(curl -w "%{http_code}" --location  --cacert ${cert_path}/ca-cert.pem --cert ${cert_path}/api-user.pem -v --request POST https://${owner_ip}:${owner_svc_port}/api/v1/owner/resource?filename=${script_file_name} --header 'Content-Type: text/plain' --data-binary "@${script_file_name}")

if [[ ${response} != "200" ]]; then
        echo "Owner resource API failed first ${response}"
        exit 1
fi

echo "Owner resource API is success ${response}"

response=$(curl -w "%{http_code}" --location  --cacert ${cert_path}/ca-cert.pem --cert ${cert_path}/api-user.pem -v --request POST https://${owner_ip}:${owner_svc_port}/api/v1/owner/resource?filename=test.sh --header 'Content-Type: text/plain' --data-binary "@test.sh")
if [[ ${response} != "200" ]]; then
        echo "Owner resource API failedseocnd ${response}"
        exit 1
fi

echo "Owner resource API is success ${response}"


# Give instruction to execute file.
script_file_name_str="\"$script_file_name\""

response=$(curl -w "%{http_code}" --cacert ${cert_path}/ca-cert.pem --cert ${cert_path}/api-user.pem -v --request POST https://${owner_ip}:${owner_svc_port}/api/v1/owner/svi --header 'Content-Type: text/plain' --data-raw "[{\"filedesc\" : ${script_file_name_str}, \"resource\" : ${script_file_name_str}},{\"exec_cb\" :[\"/bin/bash\",${script_file_name_str}]},{\"fetch\" : \"result.txt\"}]")
if [[ ${response} != "200" ]]; then
        echo "Owner svi API failed third ${response}"
        exit 1
fi

echo "Owner svi API is success ${response}"





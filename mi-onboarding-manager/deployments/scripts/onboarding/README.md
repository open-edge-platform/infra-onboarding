


## Kubernetes and Tinkerbell Installation

Update config file which holds all the configuration details needed for the setup.

```shell
$ cd platform-director/onboarding
$ vi config
```

Use setup_tinkebell_stack.sh to install tinkerbell on a host (provisioner) if not done previously.

```shell
$ cd platform-director/tinkerbell_setup_scripts
$ ./setup_tinkebell_stack.sh
```

This will install rk2 based kubernetes cluster along with tinkerbell stack customised for intel environment.

* Note : Populate the ip and interfaces in the script before run 

## FDO services installation

```shell
$ cd platform-director/onboarding/setup_scripts
$ ./setup_FDO_services.sh <server private ip>
```

This will install FDO services(mfg, rv, owner, db) to the Provisioner host.


## Tinkerbell actions installation

```shell
$ ./setup_actions.sh
```
Below is alternative when config file not used to provide pd_ip and loadbalancer_ip
```shell
$ ./setup_actions.sh <pd_ip> <loadbalancer_ip>
```


This will install all tinker actions, build hooks and store it in the local registry.

## Running Onboarding for device on Tinkerbell

Common script to setup fdo on device, run di, to and production flow.

`install_manifests_intel.sh` is common script can be used to run tinkerbell workerflows.

```shell
$ cd platform-director/onboarding/tinker_workflows
$ ./install_manifests_intel.sh "Provisioner_private_ip" "Loadbalance IP" "SUT ip" "SUT MAC" <device_setup/di/prod_(jammy/focal/bkc/focal-ms)> <disk> <CLIENT-SDK-TPM/CLIENT-SDK>
```
>> CLIENT-SDK-TPM is recomended for proper Root of trust.

> Change the Disk Name in the script as actual used ( e.g. /dev/sda )

#### Running the Manufacturer Flow

This will do pxeboot and DI flow from tinkerbell workflow.

- Run following command in terminal

```shell
$ ./install_manifests_intel.sh "Provisioner_pvt_ip" "Loadbalance IP" "SUT IP" "SUT MAC" <di> <disk> <CLIENT-SDK-TPM/CLIENT-SDK>
```

>NOTE: ./install_manifests_intel.sh "Provisioner_pvt_ip" "Loadbalance IP" "SUT IP" "SUT MAC" device_setup <disk> CLIENT-SDK-TPM can be used to know about device storage / fdo client type support , read from kubectl boot logs

-  Boot SUT in PXE mode. Ensure there is no partition present in desired Storage Disk previously selected.

- <u>Keep note of the Device serial printed to be used for ov extension</u>


- To extend the voucher do following

```shell
$ cd $HOME/pri-fidoiot/component-samples/demo/scripts
$ unset http_proxy
$ unset https_proxy
$ bash extend_upload.sh  -m sh -c ./secrets/ -e mtls -m <manufacturer ip> -o <owner ip> -s Device_serial
```
> Note : In normal configuration manufacturer ip and Owner ip are same and equal to pd_host_ip

In this step we run svi command to get the details while doing To2
```shell
$ cd <---fdo_script folder-->
$ bash svi_script.sh -c /home/$USER/pri-fidoiot/component-samples/demo/owner/secrets -o OWNERIP -p OWNER_HTTPS_PORT -s <--script to run on device side to get details-->

For example :
    $ bash svi_script.sh -c /home/intel/pri-fidoiot/component-samples/demo/owner/secrets -o 10.xx.xxx.xxx -p 8043 -s test.sh
```


#### Checking the  TO2  completion status

Run following command to know about To2 state from kubctl boot log.
>  [No need to run for deployment, will be handled by automation/owner service. Only documented here for development and testing]
> To2 will complete at deployment site , persisted hook on storage will complete TO2 without any workflow.


- in terminal 

```shell
$ cd $HOME/pri-fidoiot/component-samples/demo/scripts
$ guid_value=$(cat "${device_serial}_guid.txt")
$ response=$(curl --location --request GET "https://$pd_ip:8043/api/v1/owner/state/$guid_value" \
        --cacert /home/$USER/pri-fidoiot/component-samples/demo/owner/secrets/ca-cert.pem \
        --cert /home/$USER/pri-fidoiot/component-samples/demo/owner/secrets/api-user.pem 2>&1)

$ value=$(grep -o "\"to2CompletedOn\" : \".*\"," <<<$response | awk ' { print $3" "$4 } ')
$ echo $value

```
>  {"to2CompletedOn" : "2023-07-28 16:46:17.246","to0Expiry" : "2023-07-28 16:46:11.597"}



#### Running the Production Flow

- Run following command in terminal to install bkc/jammy/focal/focal-ms

```shell
$ ./install_manifests_intel.sh "Provisioner_pvt_ip" "Loadbalance IP" "SUT IP" "SUT MAC" prod_(jammy_focal/bkc) <disk> CLIENT-SDK-TPM

For example :
    For example :
    $ ./install_manifests_intel.sh "Provisioner_pvt_ip" "Loadbalance IP" "SUT IP" "SUT MAC" prod_jammy /dev/sda CLIENT-SDK-TPM
```


## README for PDCTL 

## To Build the PDCTL 
	1. Navigate to the `cmd/pdctl` directory.
	2. Run `go install` or Run `go build .`

## In the same path cmd/pdctl
	pdctl <commands>

## Examples of subcommands: 	
	```
	pdctl [Will give all the subcommands of pdctl and each subcommands can be explored for its own subcommands.]

	pdctl instance <commands> 

	pdctl host <commands> 

	pdctl profile <commands>
	```
## Examples of Instance Resource commands:

	1. pdctl instance create --addr=<ip or localhost>:<port> --insecure --category=2 --name=BIOS  

		(or)

	pdctl instance create --addr=<ip>:<port> --insecure --input_file="<complete-path\cmd\pdctl\commands\yaml\artifact_sample.yaml"	

	2. pdctl instance get --addr=<ip or localhost>:<port> --insecure

	3. pdctl instance delete --addr=<ip or localhost>:<port> --insecure  --category=2

	4. pdctl instance update --addr=<ip or localhost>:<port> --insecure --resource_id=<resource_id>

## Examples of Host resource commands:

	1. pdctl host add --addr=<ip or localhost>:<port> --insecure --hw-id=<hw_id>

		(or)

	pdctl host add  --addr=<ip>:<port> --insecure --input_file="<complete-path\cmd\pdctl\commands\yaml\nodes_sample.yaml"

	2. pdctl host get --addr=<ip or localhost>:<port> --insecure

	3. pdctl host update --addr=<ip or localhost>:<port> --hw-id=<hw_id> --insecure

	4. pdctl host delete --addr=<ip or localhost>:<port>  --insecure --hw-id=<hw_id>

		
## Examples of Onboarding commands:

Note: Make sure Inventory manager and Onboarding Manager is deployed and running

	1. pdctl onboard --addr=<ip>:<onboardingmgr_port> --profile-name=<profile_name> --inv_addr=<ip>:<inventorymgr_port> --insecure
        
## Examples of Instance Resources Commands:

Note: Connect to Inventory service (add parameters accordingly)

	1. pdctl instance-res create --addr=<ip or localhost>:<port> --kind=RESOURCE_KIND_INSTANCE --vm-cpu-cores=<value> --insecure

	2. pdctl instance-res get --addr=<ip or localhost>:<port> --insecure

	3. pdctl instance-res getById --addr=<ip or localhost>:<port> --insecure -r=<resource-id>

	4. pdctl instance-res update --addr=<ip or localhost>:<port> --insecure -r=<resource-id> --fields=<fields to update>

	5. pdctl instance-res delete --addr=<ip or localhost>:<port> --insecure -r=<resource-id>

## Examples of Host Resources Commands:

Note: Connect to Inventory service (add parameters accordingly)

	1. pdctl host-res create --addr=<ip or localhost>:<port> --insecure  --hostname=<name> --bmc-kind=<value> --bmc-ip=<bmc_ip> --bmc-username=<username> --bmc-password=<password> --uuid=<id>

	2. pdctl host-res get --addr=<ip or localhost>:<port> --insecure

	3. pdctl host-res getById --addr=<ip or localhost>:<port> --insecure  -r=<resource_id>

	4. pdctl host-res getByUUID --addr=<ip or localhost>:<port> --insecure -u=<uuid>

	5. pdctl host-res delete --addr=<ip or localhost>:<port> --insecure -r=<resource_id>

	6. pdctl host-res update --addr=<ip or localhost>:<port> --insecure -r=<resource_id> --bmc-username=<updated_value> 

## Examples of Mapping Instances and Host resource Commands:

	1. pdctl host-res create --addr=<ip or localhost>:<port> --insecure  --hostname=<name> --bmc-kind=<value> --bmc-ip=<bmc_ip> --bmc-username=<username> --bmc-password=<password> --uuid=<id> --sut-ip=<mgmt_ip>

	Note : Create Instance with host ID generated from above command

	2. pdctl instance-res create --addr=<ip or localhost>:<port> --kind=RESOURCE_KIND_INSTANCE --vm-cpu-cores=<value> --insecure --hostID=<host-id from above step 1>

	Note : Once the reconcliation starts, DKAM will send the response for respective resource and onboarding will be started.


## Examples of OS Resources Commands:

Note: Connect to Inventory service (add parameters accordingly)

	1. pdctl os-res create --addr=<ip or localhost>:<port> --insecure --profileName=<profilename> --repo_url=<url>

	2. pdctl os-res get --addr=<ip or localhost>:<port> --insecure

	3. pdctl os-res getById --addr=<ip or localhost>:<port> --insecure -r=<resource_id>

	4. pdctl os-res delete --addr=<ip or localhost>:<port> --insecure -r=<resource_id>

	5. pdctl os-res update --addr=<ip or localhost>:<port> --insecure -r=<resource_id> -a=<value> -k=<command> -l=<url value> 


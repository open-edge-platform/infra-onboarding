# Onboarding Manager Basic API

## Pre-requisite
1. GOLANG installed
2. Inventory manager Running
2. Dkam manager Running

## 1. Enabling GOLANG
```
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin
export GOPATH=$(go env GOPATH)
export MGR_HOST=localhost/IP
export ONBMGR_PORT=50054
```
### 1.1 Exporting Onboarding Parameters
```
export PD_IP=<pd_ip>
export DISK_PARTITION=/da/sda
export LOAD_BALANCER_IP=<load_balancer_ip>
export IMAGE_TYPE= prod_bkc
```
## 2. Run Onboarding manager 
```
go run main.go
```
### 2.1 Script to Run the Onboarding manager
```
    1. go to infrastructure.edge.iaas.platform\scripts\edge-iaas-platform\platform-director\onboarding\setup_scripts
    2. chmod +x onboardingsetup.sh
    3. ./onboardingsetup.sh <MGR_HOST:localhost/ip> <ONBMGR_PORT>

Note: This will start the server please use the another terminal to trigger pdctl command 
```
### 2.2 Manual running the Onboarding manager
``` 
    1. check all the Pre-requisite mentioned above
    2. Enabling GOLANG by exporting all variables 
    3. run main.go to start onboarding grpc server
        cd cmd/onboardingmgr/
        go run main.go
```
## 3. Build and Test for MS and BKC
```
How to run Onboaridng manager and trigger command from pdctl to start onboarding for the profile 

step 1) Navigate to the 'frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service` directory

step 3) check if onboarding manager and inventory service  is running 

```
### 3.1 Pdctl for End to End flow
````
    1) Navigate to frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/cmd/pdctl/
    
    2) Create a Host resource 

    3) Create a OS resource

    4) Create a Instance resource and associate it with host-id and os-id

    5) Once onboarding manager is running, it will reconcile with the Instance state and onboarding process will start.

    Examples:

        1.  pdctl host-res create --addr=localhost:50051 --insecure  --hostname=OS  --bmc-kind=BAREMETAL_CONTROLLER_KIND_PDU  --uuid=9fa8a788-f9f8-434a-8620-bbed2a12b0ad -t=10.49.76.113 -x=1c:69:7a:a8:12:af -c=INSTANCE_STATE_UNSPECIFIED --bmc-ip=10.223.87.65
    
        2. pdctl os-res create --addr=localhost:50051 --profileName="osprofile"  -l="repo_url" --insecure

        3. pdctl host-res get --addr=localhost:50051 --insecure // Get the host id to associate it with instance

        4. pdctl os-res get --addr=localhost:50051 --insecure // Get the os id to associate it with instance

        5. pdctl instance-res create --addr=localhost:50051 --insecure --kind=RESOURCE_KIND_INSTANCE --osID=<os-id from step 4> --hostID=<host-id from step 3>
 

Note: make sure your dkam is running to get the urls 
      Make sure your inventory service is running mentioned in step 5.1

````
## 4. Limitation and Caveats :
    Limitation :
        1) we can now only run one worklow for per profile either MS or BKC for multiple devices 
        2) let one profile onboarding should be finished then only trigger other profile command 

    Caveats :
		1) Ensure to set MGR_HOST and ONBMGR_PORT with desired IP address and port address to run this application hassle-free

		2) Ensure `startonboarding: true` in
		infrastructure.edge.iaas.platform/cmd/pdctl/commands/yaml/profile_sample.yaml to receive the start onboarding notification onboardingmgr

		4) Always ensure 'pdctl artifact get --addr=localhost:50052 --insecure' gives artifact output before issuing
		`pdctl profile create`

        5)Make sure to  add all the profile details in profile sample.yaml /cmd/pdctl


===============================================================================================================================
===============================================================================================================================

# 5. Test Node Operations : Create, Get, Update, and Delete nodes using gRPC client

## 5.1 Run the Edge Orchestration Inventory Service
```
git clone https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory

git checkout 2936bdb169aa05d5e09236b6c304d54fad80eedc

cd frameworks.edge.one-intel-edge.maestro-infra.services.inventory/

make go-build

If the db is already running, stop the it,
make db-stop 

export PGUSER=admin
export PGHOST=localhost
export PGDATABASE=postgres
export PGPORT=5432
export PGPASSWORD=pass
export PGSSLMODE=disable

curl -sSf https://atlasgo.sh | sh

sudo rm -rf /usr/share/migrations/
sudo cp -avr internal/ent/migrate/migrations /usr/share/

make db-start

./build/miinv --policyBundlePath=./build/policy_bundle.tar.gz
```

## 5.2 Run the Onboarding service
```
git clone https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service

cd  frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service
go run cmd/onboardingmgr/main.go
```

## 5.3 Run the PDCTL command, client to the onboarding service
```
NOTE : 1) Serial number is the unique number used to create the host number.
       2)  Ensure to pass serial number as a mandatory argument in order to 
           perform any crud operation. If any node exists with a given serial
           number then the system reflects with a message "Node already exists"
       3)  Create operation throws an error saying "Node already exists", when
           same serial number is repeatedly used to create a node.
       4)  Delete operation is not going to delete the node completely from the database,
           it is going to set the "current_state" as "HOST_STATE_DELETED"
        
Create operation command :
-------------------------
go  run cmd/pdctl/main.go host add --addr=localhost:50054 --insecure --hw-id=123 --mac=ab:cd:ef:12:34:56 --sutip=192.168.1.654 --serial-number=98330 --uuid="14921492-1492-1492-1492-123412341249" --bmc-ip="192.168.0.125" --bmc-interface=true --host-nic-dev-name="eth0"

Read operation command :
-----------------------
go run cmd/pdctl/main.go host get --addr=localhost:50054 --insecure --hw-id=123 --serial-number=98328 --uuid="14921492-1492-1492-1492-123412341249"

Update operation command : update mac, sutip, and bmc-ip
-------------------------
go run cmd/pdctl/main.go host update --addr=localhost:50054 --insecure --hw-id=123 --mac=ab:cd:ef:12:34:56 --sutip=192.168.1.654 --serial-number=98330 --uuid="14921492-1492-1492-1492-123412341249" --bmc-ip="192.168.0.125" --bmc-interface=true --host-nic-dev-name="eth0"

Delete operation command :
-------------------------
go run cmd/pdctl/main.go host delete --addr=localhost:50054 --insecure --hw-id=123 --serial-number=98328 --uuid="14921492-1492-1492-1492-123412341249"

```

### End to End flow with PDCTL

```mermaid
sequenceDiagram
%%{wrap}%%
  autonumber
  actor User as Trusted SI User/PDCTL
  box NavajoWhite FMaaS
    participant DKAM as DKAM
    participant OM as Onboarding Manager
    participant IM as Inventory Service
  end
  box Edge Node
    participant Node as Node
  end
  note over IM,User: PDCTL registers as Inventory Client.Creates Host & associates Host to OS instance. 
  User ->> IM : Create Host with Serial Number,UUID,MAC & IP
  IM ->> User : Return unique Resource ID for the Host
  User ->> IM : Create Instance Resource with Resource ID of Host
  note over OM,IM : Inventory Service issues a Intent towards onboarding manager(Registered as Resource manager) after Instance is created.
  IM ->>+ OM : Report when Instance Desired state - Onboarding Success
  OM ->>+IM : GetInstancebyResourceID(ID) 
  IM ->>+ OM : Return Instance resource with Host ID
  OM ->>+ IM : GetHostDetails(Host ID)
  OM ->>+ DKAM : GetOSDetails(Profile)
  DKAM ->>+ OM : Return manifest.yaml with OS & Overlays
  OM ->>+ OM : StartOnboarding(Profile + HW Details)
  note over OM,IM: Onboarding process continues to FDO & OS Provisioning 
```

### Opens

1. Need to create OS resource as a first step & populate with the manifest from DKAM. 
2. While creating instance resource, OS resource also needs to be associated along with Host Resource.
3. Integration with CDN boots.

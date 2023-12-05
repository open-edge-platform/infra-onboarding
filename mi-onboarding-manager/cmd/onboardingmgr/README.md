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

step 1) Navigate to the 'infrastructure.edge.iaas.platform` directory

step 3) check if onboaridng manager and inventory manager  is running 

```
### 3.1. To Run MS Workflow
````
    step 1) change the profil_sample.yaml  present in /cmd/pdctl/commands/yaml/profile_sample.yaml
        1. Add Hw serial id 
        2. Add HW macid 
            note: hw mac format should be ip-mac  10.xx.xx.113-1c:xx:7a:xx:xx 
        3. Add pd ip ,loadbalacer ip,diskpartition 
        4. For ms workflow just define prod_focal-ms in platformtype
        5. For zeroTouch/Non zero touch ZT/NZT in env
        6. make the start  startonboarding: true to start onbaording
    step 2) Trigger Pdctl commands
           1. create the articate  pdctl artifact create --addr=localhost:50052 --insecure --input_file=./cmd/pdctl/commands/yaml/artifact_sample.yaml
           2. replace the osartid with above genrated artifact value under ubuntu similarly for fwartids under bios 
           3. Run pdctl profile create command to create the profile
                pdctl profile create --addr=inventorymgrip:<port to communicate> --insecure --input_file=./commands/yaml/profile_sample.yaml
           4. Run pdctl onboard command
                pdctl onboard --addr=onboardingmgrip:<port>--profile-name=<Name used in profile_sample> --inv_addr=inventorymgrip:<port to communicate>  --insecure

            
       Example:
        pdctl artifact create --addr=localhost:31846 --insecure --input_file=./cmd/pdctl/commands/yaml/artifact_sample.yaml
        pdctl profile create --addr=localhost:31846 --insecure --input_file=./cmd/pdctl/commands/yaml/profile_sample.yaml
        pdctl onboard --addr=onboardingmgrip:<port>--profile-name=<Name used in profile_sample> --inv_addr=inventorymgrip:<port to communicate>  --insecure`

```
### 3.2. To Run BKC Workflow
````
    step 1) change the profil_sample.yaml  present in /cmd/pdctl/commands/yaml/profile_sample.yaml
        1. Add Hw serial id 
        2. Add HW macid 
            note: hw mac format should be ip-mac  10.xx.xx.113-1c:xx:7a:xx:xx 
        3. Add pd ip ,loadbalacer ip,diskpartition 
        4. For ms workflow just define prod_bkc in platformtype
        5. For zeroTouch/Non zero touch ZT/NZT in env
        6. make the start  startonboarding: true to start onbaording
    step 2) Trigger Pdctl commands
           1. create the articate  pdctl artifact create --addr=localhost:50052 --insecure --input_file=./cmd/pdctl/commands/yaml/artifact_sample.yaml
           2. replace the osartid with above genrated artifact value under ubuntu similarly for fwartids under bios 
           3. Run pdctl profile create command to create the profile
                pdctl profile create --addr=inventorymgrip:<port to communicate> --insecure --input_file=./commands/yaml/profile_sample.yaml
           4. Run pdctl onboard command
                pdctl onboard --addr=onboardingmgrip:<port>--profile-name=<Name used in profile_sample> --inv_addr=inventorymgrip:<port to communicate>  --insecure

            
       Example:
        pdctl artifact create --addr=localhost:31846 --insecure --input_file=./cmd/pdctl/commands/yaml/artifact_sample.yaml
        pdctl profile create --addr=localhost:31846 --insecure --input_file=./cmd/pdctl/commands/yaml/profile_sample.yaml
        pdctl onboard --addr=onboardingmgrip:<port>--profile-name=<Name used in profile_sample> --inv_addr=inventorymgrip:<port to communicate>  --insecure`

Note: make sure your dkam is running to get the urls 

```
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


====================================================================================================================================================
====================================================================================================================================================

# 5. Test Node Operations : Create, Get, Update, and Delete nodes using gRPC client

## 5.1 Run the Maestro Inventory Service
```
git clone https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory

cd frameworks.edge.one-intel-edge.maestro-infra.services.inventory/

make go-build

make db-start

export PGUSER=admin
export PGHOST=localhost
export PGDATABASE=postgres
export PGPORT=5432
export PGPASSWORD=pass
export PGSSLMODE=disable

curl -sSf https://atlasgo.sh | sh

sudo cp -avr internal/ent/migrate/migrations /usr/share/

./build/miinv --policyBundle=./build/policy_bundle.tar.gz
```

## 5.2 Run the Onboarding service
```
git clone https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service

go run cmd/onboardingmgr/main.go
```

## 5.3 Run the PDCTL command, client to the onboarding service
```
go run cmd/pdctl/main.go nodes add --addr=localhost:50052 --insecure --hw-id=123

go run cmd/pdctl/main.go nodes delete --addr=localhost:50052 --insecure --hw-id=123
```

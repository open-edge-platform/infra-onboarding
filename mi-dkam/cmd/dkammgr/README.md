# DKAM Manager API

### Pre-requisite
GOLANG installed 

### 1. Enabling GOLANG
```
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin
export GOPATH=$(go env GOPATH)
```
### 2. Run DKAM Manager
```
cd ./cmd/dkammgr
go run ./main.go
```
If succeed, stdout will looks like below:
```
Starting gRPC server on port :5581
```
### 3. Open 2nd terminal and execute following:
```
cd ./internal/dkammgr/test/client
go run ./main.go
```
If succeed, stdout will looks like below:
```
2023/12/14 12:06:10 GetArtifacts.
2023/12/14 12:06:10 Result: manifest_file:"osurl: https://af01p-png.devtools.intel.com/artifactory/hspe-edge-png-local/ubuntu-base/20230911-1844/default/ubuntu-22.04-desktop-amd64+intel-iot-37-custom.qcow2.bz2\noverlayscripturl: https://ubit-artifactory-sh.intel.com/artifactory/sed-dgn-local/yocto/dev-test-image/DKAM/IAAS/ADL/installer23WW44.4_2148.sh\n"  statusCode:true

```
### 3. DKAM helm chart deployment in Maestro:
```
Helm chart deployment in Maestro
export AZUREAD_USER="youremail"
export AZUREAD_PASS="pass"
export AZUREAD_CLIENTID="clientid"
mage deploy:kindAll

Deploy DKAM helm chart in dev instance
helm install dkam --set env.MODE="dev" --set env.SeverUrl="server_url"
 


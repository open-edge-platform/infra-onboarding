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
### 3. DKAM helm chart deployment:
```
Helm chart deployment
export AZUREAD_USER="azure_ad_token@intel.com"
export AZUREAD_PASS="MysoreKarnataka@570011"
export AZUREAD_CLIENTID="4ee465a2-6805-425d-a5a0-9ccf938fd38d"
mage deploy:kindAll

Deploy DKAM helm chart in dev instance
cd ~frameworks.edge.one-intel-edge.maestro-infra.charts/mi-dkam
helm install dkam --set env.mode="dev" --set traefikReverseProxy.dnsname="https://tink.dkam.jf.intel.com/tink-stack" --set traefikReverseProxy.proxyip="10.49.1.24" -n <namespace>.
 


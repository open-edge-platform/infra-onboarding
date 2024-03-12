# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [onboarding.proto](#onboarding-proto)
    - [ArtifactData](#onboardingmgr-ArtifactData)
    - [ArtifactRequest](#onboardingmgr-ArtifactRequest)
    - [ArtifactResponse](#onboardingmgr-ArtifactResponse)
    - [CustomerParams](#onboardingmgr-CustomerParams)
    - [HwData](#onboardingmgr-HwData)
    - [NodeData](#onboardingmgr-NodeData)
    - [NodeRequest](#onboardingmgr-NodeRequest)
    - [NodeResponse](#onboardingmgr-NodeResponse)
    - [Ports](#onboardingmgr-Ports)
    - [Proxy](#onboardingmgr-Proxy)
    - [SecureBootResponse](#onboardingmgr-SecureBootResponse)
    - [SecureBootStatRequest](#onboardingmgr-SecureBootStatRequest)
    - [Supplier](#onboardingmgr-Supplier)
  
    - [ArtifactData.ArtifactCategory](#onboardingmgr-ArtifactData-ArtifactCategory)
    - [ArtifactData.Response](#onboardingmgr-ArtifactData-Response)
    - [NodeData.Response](#onboardingmgr-NodeData-Response)
    - [SecureBootResponse.Status](#onboardingmgr-SecureBootResponse-Status)
    - [SecureBootStatRequest.Status](#onboardingmgr-SecureBootStatRequest-Status)
  
    - [NodeArtifactServiceNB](#onboardingmgr-NodeArtifactServiceNB)
    - [OnBoardingSB](#onboardingmgr-OnBoardingSB)
  
- [Scalar Value Types](#scalar-value-types)



<a name="onboarding-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## onboarding.proto



<a name="onboardingmgr-ArtifactData"></a>

### ArtifactData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the artifact |
| version | [string](#string) |  | Version of the artifact |
| platform | [string](#string) |  | Platform of the artifact |
| category | [ArtifactData.ArtifactCategory](#onboardingmgr-ArtifactData-ArtifactCategory) |  | Category of the artifact ex:BIOS,OS etc., |
| description | [string](#string) |  | Description of the artifact |
| details | [Supplier](#onboardingmgr-Supplier) |  | Supplier details |
| package_url | [string](#string) |  | URL of the package |
| author | [string](#string) |  | Author of package |
| state | [bool](#bool) |  | state |
| license | [string](#string) |  | License information |
| vendor | [string](#string) |  | vendor details |
| manufacturer | [string](#string) |  | manufacter details |
| release_data | [string](#string) |  | Release data |
| artifact_id | [string](#string) |  | Artifact ID generated while creating an artifact. This can be zero if not available during CreateArtifact Call or Batch actions like DeleteAll. |
| result | [ArtifactData.Response](#onboardingmgr-ArtifactData-Response) |  |  |






<a name="onboardingmgr-ArtifactRequest"></a>

### ArtifactRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-ArtifactData) | repeated | Payload data represented as an array or list |






<a name="onboardingmgr-ArtifactResponse"></a>

### ArtifactResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-ArtifactData) | repeated | Payload data {will be same as request for CREATE/DELETE}. |






<a name="onboardingmgr-CustomerParams"></a>

### CustomerParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dps_scope_id | [string](#string) |  | DPS Scope ID |
| dps_registration_id | [string](#string) |  | DPS registration ID |
| dps_enrollment_sym_key | [string](#string) |  | DPS Enrollment Symetric Key |






<a name="onboardingmgr-HwData"></a>

### HwData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hw_id | [string](#string) |  | HW ID of Node |
| mac_id | [string](#string) |  | Mac ID of Node |
| sut_ip | [string](#string) |  | sutip |
| cus_params | [CustomerParams](#onboardingmgr-CustomerParams) |  | Azure Specific Parameters |
| disk_partition | [string](#string) |  | Disk Partition Details |
| platform_type | [string](#string) |  | Device platform type |
| serialnum | [string](#string) |  |  |
| uuid | [string](#string) |  |  |
| bmc_ip | [string](#string) |  |  |
| bmc_interface | [bool](#bool) |  |  |
| host_nic_dev_name | [string](#string) |  |  |
| SecurityFeature | [uint32](#uint32) |  |  |






<a name="onboardingmgr-NodeData"></a>

### NodeData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hw_id | [string](#string) |  | HW Node ID |
| platform_type | [string](#string) |  | Platform details of the node //EHL,ADL/RPL/etc., |
| fw_artifact_id | [string](#string) |  | Node FW Artifact ID to be stored here.This ID is retured to GetArtifacts{id} |
| os_artifact_id | [string](#string) |  | Node OS Artifact ID to be stored here.This ID is retured to GetArtifacts{id} |
| app_artifact_id | [string](#string) |  | TODO: a new member for Image artifact has to be added here, for now, app_artifact_id is used for image artifact

Node App Artifact ID to be stored here.This ID is retured to GetArtifacts{id} |
| plat_artifact_id | [string](#string) |  | Node Platform Artifact ID to be stored here.This ID is retured to GetArtifacts{id} |
| device_type | [string](#string) |  | Node can be physical or virtual or container. If ID is not given, then all nodes FW artifacts wil be returned |
| device_info_agent | [string](#string) |  | Inventory Agent update SBOM &amp; HBOM details during bootup. |
| device_status | [string](#string) |  | Only Inventory Agent Update READY Status to Inventory Manager. Other status by Admin or other managers UNCLAIMED,CLAIMED,READY,MAINTENANCE,ERROR,DECOMMISSIONED |
| update_status | [string](#string) |  | Update Manager Update the Node update status. |
| update_available | [string](#string) |  | Update manager updates if update is available. |
| onboarding_status | [string](#string) |  | Onboarding Status |
| node_id | [string](#string) |  | Generated Node ID. This field can be left empty for Create or DeleteAll |
| result | [NodeData.Response](#onboardingmgr-NodeData-Response) |  | Result |
| hwdata | [HwData](#onboardingmgr-HwData) | repeated |  |






<a name="onboardingmgr-NodeRequest"></a>

### NodeRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-NodeData) | repeated | Payload data |






<a name="onboardingmgr-NodeResponse"></a>

### NodeResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-NodeData) | repeated | Payload data |






<a name="onboardingmgr-Ports"></a>

### Ports



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inv_mgr_port | [string](#string) |  | inventory manager port |
| up_mgr_port | [string](#string) |  | update manager port |
| oob_mgr_port | [string](#string) |  | oob manager port |
| tele_mgr_port | [string](#string) |  | Telemetry manager port |






<a name="onboardingmgr-Proxy"></a>

### Proxy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_proxy | [string](#string) |  | http proxy |
| https_proxy | [string](#string) |  | http proxy |
| no_proxy | [string](#string) |  | http proxy |
| socks_proxy | [string](#string) |  | socks proxy |
| rsync_proxy | [string](#string) |  | rsync proxy |






<a name="onboardingmgr-SecureBootResponse"></a>

### SecureBootResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| result | [SecureBootResponse.Status](#onboardingmgr-SecureBootResponse-Status) |  |  |
| guid | [string](#string) |  |  |






<a name="onboardingmgr-SecureBootStatRequest"></a>

### SecureBootStatRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| result | [SecureBootStatRequest.Status](#onboardingmgr-SecureBootStatRequest-Status) |  |  |
| guid | [string](#string) |  | GUID |






<a name="onboardingmgr-Supplier"></a>

### Supplier



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of supplier |
| url | [string](#string) |  | URL of supplier |
| contact | [string](#string) |  | Contact details of supplier |





 


<a name="onboardingmgr-ArtifactData-ArtifactCategory"></a>

### ArtifactData.ArtifactCategory


| Name | Number | Description |
| ---- | ------ | ----------- |
| DEFAULT | 0 | Setting default artifact type getting all artifact |
| BIOS | 1 | BIOS Artifact |
| OS | 2 | OS Artifact |
| APPLICATION | 3 | Application Artifact |
| IMAGE | 4 | Container image Artifact |
| PLATFORM | 5 | Type of platform of the artifact |



<a name="onboardingmgr-ArtifactData-Response"></a>

### ArtifactData.Response


| Name | Number | Description |
| ---- | ------ | ----------- |
| SUCCESS | 0 | Success |
| FAILURE | 1 | Failure |



<a name="onboardingmgr-NodeData-Response"></a>

### NodeData.Response


| Name | Number | Description |
| ---- | ------ | ----------- |
| SUCCESS | 0 | Success |
| FAILURE | 1 | Failure |



<a name="onboardingmgr-SecureBootResponse-Status"></a>

### SecureBootResponse.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| SUCCESS | 0 | Success |
| FAILURE | 1 | Failure |



<a name="onboardingmgr-SecureBootStatRequest-Status"></a>

### SecureBootStatRequest.Status


| Name | Number | Description |
| ---- | ------ | ----------- |
| SUCCESS | 0 | Success |
| FAILURE | 1 | Failure |


 

 


<a name="onboardingmgr-NodeArtifactServiceNB"></a>

### NodeArtifactServiceNB
Artifact &amp; Node Endpoints towards Inventory Manager

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateArtifacts | [ArtifactRequest](#onboardingmgr-ArtifactRequest) | [ArtifactResponse](#onboardingmgr-ArtifactResponse) |  |
| GetArtifacts | [ArtifactRequest](#onboardingmgr-ArtifactRequest) | [ArtifactResponse](#onboardingmgr-ArtifactResponse) |  |
| UpdateArtifactsById | [ArtifactRequest](#onboardingmgr-ArtifactRequest) | [ArtifactResponse](#onboardingmgr-ArtifactResponse) |  |
| DeleteArtifacts | [ArtifactRequest](#onboardingmgr-ArtifactRequest) | [ArtifactResponse](#onboardingmgr-ArtifactResponse) |  |
| CreateNodes | [NodeRequest](#onboardingmgr-NodeRequest) | [NodeResponse](#onboardingmgr-NodeResponse) |  |
| GetNodes | [NodeRequest](#onboardingmgr-NodeRequest) | [NodeResponse](#onboardingmgr-NodeResponse) |  |
| UpdateNodes | [NodeRequest](#onboardingmgr-NodeRequest) | [NodeResponse](#onboardingmgr-NodeResponse) |  |
| DeleteNodes | [NodeRequest](#onboardingmgr-NodeRequest) | [NodeResponse](#onboardingmgr-NodeResponse) |  |


<a name="onboardingmgr-OnBoardingSB"></a>

### OnBoardingSB


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SecureBootStatus | [SecureBootStatRequest](#onboardingmgr-SecureBootStatRequest) | [SecureBootResponse](#onboardingmgr-SecureBootResponse) | updates secureboot BIOS status in Edge Node |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |


# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [onboarding.proto](#onboarding-proto)
    - [ArtifactData](#onboardingmgr-v1-ArtifactData)
    - [CreateArtifactsRequest](#onboardingmgr-v1-CreateArtifactsRequest)
    - [CreateArtifactsResponse](#onboardingmgr-v1-CreateArtifactsResponse)
    - [CreateNodesRequest](#onboardingmgr-v1-CreateNodesRequest)
    - [CreateNodesResponse](#onboardingmgr-v1-CreateNodesResponse)
    - [CustomerParams](#onboardingmgr-v1-CustomerParams)
    - [DeleteArtifactsRequest](#onboardingmgr-v1-DeleteArtifactsRequest)
    - [DeleteArtifactsResponse](#onboardingmgr-v1-DeleteArtifactsResponse)
    - [DeleteNodesRequest](#onboardingmgr-v1-DeleteNodesRequest)
    - [DeleteNodesResponse](#onboardingmgr-v1-DeleteNodesResponse)
    - [GetArtifactsRequest](#onboardingmgr-v1-GetArtifactsRequest)
    - [GetArtifactsResponse](#onboardingmgr-v1-GetArtifactsResponse)
    - [GetNodesRequest](#onboardingmgr-v1-GetNodesRequest)
    - [GetNodesResponse](#onboardingmgr-v1-GetNodesResponse)
    - [HwData](#onboardingmgr-v1-HwData)
    - [NodeData](#onboardingmgr-v1-NodeData)
    - [Ports](#onboardingmgr-v1-Ports)
    - [Proxy](#onboardingmgr-v1-Proxy)
    - [Supplier](#onboardingmgr-v1-Supplier)
    - [UpdateArtifactsByIdRequest](#onboardingmgr-v1-UpdateArtifactsByIdRequest)
    - [UpdateArtifactsByIdResponse](#onboardingmgr-v1-UpdateArtifactsByIdResponse)
    - [UpdateNodesRequest](#onboardingmgr-v1-UpdateNodesRequest)
    - [UpdateNodesResponse](#onboardingmgr-v1-UpdateNodesResponse)
  
    - [ArtifactData.ArtifactCategory](#onboardingmgr-v1-ArtifactData-ArtifactCategory)
    - [ArtifactData.Response](#onboardingmgr-v1-ArtifactData-Response)
    - [NodeData.Response](#onboardingmgr-v1-NodeData-Response)
  
    - [NodeArtifactNBService](#onboardingmgr-v1-NodeArtifactNBService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="onboarding-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## onboarding.proto



<a name="onboardingmgr-v1-ArtifactData"></a>

### ArtifactData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of the artifact |
| version | [string](#string) |  | Version of the artifact |
| platform | [string](#string) |  | Platform of the artifact |
| category | [ArtifactData.ArtifactCategory](#onboardingmgr-v1-ArtifactData-ArtifactCategory) |  | Category of the artifact ex:BIOS,OS etc., |
| description | [string](#string) |  | Description of the artifact |
| details | [Supplier](#onboardingmgr-v1-Supplier) |  | Supplier details |
| package_url | [string](#string) |  | URL of the package |
| author | [string](#string) |  | Author of package |
| state | [bool](#bool) |  | state |
| license | [string](#string) |  | License information |
| vendor | [string](#string) |  | vendor details |
| manufacturer | [string](#string) |  | manufacter details |
| release_data | [string](#string) |  | Release data |
| artifact_id | [string](#string) |  | Artifact ID generated while creating an artifact. This can be zero if not available during CreateArtifact Call or Batch actions like DeleteAll. |
| result | [ArtifactData.Response](#onboardingmgr-v1-ArtifactData-Response) |  |  |






<a name="onboardingmgr-v1-CreateArtifactsRequest"></a>

### CreateArtifactsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-CreateArtifactsResponse"></a>

### CreateArtifactsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-CreateNodesRequest"></a>

### CreateNodesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |






<a name="onboardingmgr-v1-CreateNodesResponse"></a>

### CreateNodesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |






<a name="onboardingmgr-v1-CustomerParams"></a>

### CustomerParams



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| dps_scope_id | [string](#string) |  | DPS Scope ID |
| dps_registration_id | [string](#string) |  | DPS registration ID |
| dps_enrollment_sym_key | [string](#string) |  | DPS Enrollment Symetric Key |






<a name="onboardingmgr-v1-DeleteArtifactsRequest"></a>

### DeleteArtifactsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-DeleteArtifactsResponse"></a>

### DeleteArtifactsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-DeleteNodesRequest"></a>

### DeleteNodesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |






<a name="onboardingmgr-v1-DeleteNodesResponse"></a>

### DeleteNodesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |






<a name="onboardingmgr-v1-GetArtifactsRequest"></a>

### GetArtifactsRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-GetArtifactsResponse"></a>

### GetArtifactsResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-GetNodesRequest"></a>

### GetNodesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |






<a name="onboardingmgr-v1-GetNodesResponse"></a>

### GetNodesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |






<a name="onboardingmgr-v1-HwData"></a>

### HwData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| hw_id | [string](#string) |  | HW ID of Node |
| mac_id | [string](#string) |  | Mac ID of Node |
| sut_ip | [string](#string) |  | sutip |
| cus_params | [CustomerParams](#onboardingmgr-v1-CustomerParams) |  | Azure Specific Parameters |
| disk_partition | [string](#string) |  | Disk Partition Details |
| platform_type | [string](#string) |  | Device platform type |
| serialnum | [string](#string) |  |  |
| uuid | [string](#string) |  |  |
| bmc_ip | [string](#string) |  |  |
| bmc_interface | [bool](#bool) |  |  |
| host_nic_dev_name | [string](#string) |  |  |
| security_feature | [uint32](#uint32) |  |  |






<a name="onboardingmgr-v1-NodeData"></a>

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
| result | [NodeData.Response](#onboardingmgr-v1-NodeData-Response) |  | Result |
| hwdata | [HwData](#onboardingmgr-v1-HwData) | repeated |  |






<a name="onboardingmgr-v1-Ports"></a>

### Ports



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| inv_mgr_port | [string](#string) |  | inventory manager port |
| up_mgr_port | [string](#string) |  | update manager port |
| oob_mgr_port | [string](#string) |  | oob manager port |
| tele_mgr_port | [string](#string) |  | Telemetry manager port |






<a name="onboardingmgr-v1-Proxy"></a>

### Proxy



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| http_proxy | [string](#string) |  | http proxy |
| https_proxy | [string](#string) |  | http proxy |
| no_proxy | [string](#string) |  | http proxy |
| socks_proxy | [string](#string) |  | socks proxy |
| rsync_proxy | [string](#string) |  | rsync proxy |






<a name="onboardingmgr-v1-Supplier"></a>

### Supplier



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name of supplier |
| url | [string](#string) |  | URL of supplier |
| contact | [string](#string) |  | Contact details of supplier |






<a name="onboardingmgr-v1-UpdateArtifactsByIdRequest"></a>

### UpdateArtifactsByIdRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-UpdateArtifactsByIdResponse"></a>

### UpdateArtifactsByIdResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [ArtifactData](#onboardingmgr-v1-ArtifactData) | repeated |  |






<a name="onboardingmgr-v1-UpdateNodesRequest"></a>

### UpdateNodesRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |






<a name="onboardingmgr-v1-UpdateNodesResponse"></a>

### UpdateNodesResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| payload | [NodeData](#onboardingmgr-v1-NodeData) | repeated |  |





 


<a name="onboardingmgr-v1-ArtifactData-ArtifactCategory"></a>

### ArtifactData.ArtifactCategory


| Name | Number | Description |
| ---- | ------ | ----------- |
| ARTIFACT_CATEGORY_DEFAULT_UNSPECIFIED | 0 | Setting default artifact type getting all artifact |
| ARTIFACT_CATEGORY_BIOS | 1 | BIOS Artifact |
| ARTIFACT_CATEGORY_OS | 2 | OS Artifact |
| ARTIFACT_CATEGORY_APPLICATION | 3 | Application Artifact |
| ARTIFACT_CATEGORY_IMAGE | 4 | Container image Artifact |
| ARTIFACT_CATEGORY_PLATFORM | 5 | Type of platform of the artifact |



<a name="onboardingmgr-v1-ArtifactData-Response"></a>

### ArtifactData.Response


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESPONSE_SUCCESS_UNSPECIFIED | 0 | Success |
| RESPONSE_FAILURE | 1 | Failure |



<a name="onboardingmgr-v1-NodeData-Response"></a>

### NodeData.Response


| Name | Number | Description |
| ---- | ------ | ----------- |
| RESPONSE_SUCCESS_UNSPECIFIED | 0 | Success |
| RESPONSE_FAILURE | 1 | Failure |


 

 


<a name="onboardingmgr-v1-NodeArtifactNBService"></a>

### NodeArtifactNBService
Artifact &amp; Node Endpoints towards Inventory Manager

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateArtifacts | [CreateArtifactsRequest](#onboardingmgr-v1-CreateArtifactsRequest) | [CreateArtifactsResponse](#onboardingmgr-v1-CreateArtifactsResponse) |  |
| GetArtifacts | [GetArtifactsRequest](#onboardingmgr-v1-GetArtifactsRequest) | [GetArtifactsResponse](#onboardingmgr-v1-GetArtifactsResponse) |  |
| UpdateArtifactsById | [UpdateArtifactsByIdRequest](#onboardingmgr-v1-UpdateArtifactsByIdRequest) | [UpdateArtifactsByIdResponse](#onboardingmgr-v1-UpdateArtifactsByIdResponse) |  |
| DeleteArtifacts | [DeleteArtifactsRequest](#onboardingmgr-v1-DeleteArtifactsRequest) | [DeleteArtifactsResponse](#onboardingmgr-v1-DeleteArtifactsResponse) |  |
| CreateNodes | [CreateNodesRequest](#onboardingmgr-v1-CreateNodesRequest) | [CreateNodesResponse](#onboardingmgr-v1-CreateNodesResponse) |  |
| GetNodes | [GetNodesRequest](#onboardingmgr-v1-GetNodesRequest) | [GetNodesResponse](#onboardingmgr-v1-GetNodesResponse) |  |
| UpdateNodes | [UpdateNodesRequest](#onboardingmgr-v1-UpdateNodesRequest) | [UpdateNodesResponse](#onboardingmgr-v1-UpdateNodesResponse) |  |
| DeleteNodes | [DeleteNodesRequest](#onboardingmgr-v1-DeleteNodesRequest) | [DeleteNodesResponse](#onboardingmgr-v1-DeleteNodesResponse) |  |

 



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


# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [onboarding.proto](#onboarding-proto)
    - [HwData](#onboardingmgr-HwData)
    - [NodeData](#onboardingmgr-NodeData)
    - [NodeRequest](#onboardingmgr-NodeRequest)
    - [NodeResponse](#onboardingmgr-NodeResponse)
    - [OnboardStreamRequest](#onboardingmgr-OnboardStreamRequest)
    - [OnboardStreamResponse](#onboardingmgr-OnboardStreamResponse)
  
    - [OnboardStreamResponse.NodeState](#onboardingmgr-OnboardStreamResponse-NodeState)
  
    - [InteractiveOnboardingService](#onboardingmgr-InteractiveOnboardingService)
    - [NonInteractiveOnboardingService](#onboardingmgr-NonInteractiveOnboardingService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="onboarding-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## onboarding.proto



<a name="onboardingmgr-HwData"></a>

### HwData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uuid | [string](#string) |  |  |
| serialnum | [string](#string) |  |  |
| mac_id | [string](#string) |  | Mac ID of Edge Node |
| sut_ip | [string](#string) |  | sutip |






<a name="onboardingmgr-NodeData"></a>

### NodeData



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
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
| project_id | [string](#string) |  | The project_id associated with the Edge Node, identifying the project to which the Edge Node belongs |






<a name="onboardingmgr-OnboardStreamRequest"></a>

### OnboardStreamRequest
OnboardStreamRequest represents a request sent from Edge Node to the Onboarding Manager


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| uuid | [string](#string) |  | The UUID of the Edge Node being onboarded |
| serialnum | [string](#string) |  | The serial number of the Edge Node |
| mac_id | [string](#string) |  | The MAC ID of the Edge Node |
| host_ip | [string](#string) |  | The IP (IPv4 pattern) of the Edge Node |






<a name="onboardingmgr-OnboardStreamResponse"></a>

### OnboardStreamResponse
OnboardStreamResponse represents a response sent from the Onboarding Manager to a Edge Node
over the bidirectional stream


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| status | [google.rpc.Status](#google-rpc-Status) |  | The status of the onboarding request |
| node_state | [OnboardStreamResponse.NodeState](#onboardingmgr-OnboardStreamResponse-NodeState) |  | The current state of the device as stored in Infra Inventory |
| client_id | [string](#string) |  | The client_id provided to the node upon successful onboarding |
| client_secret | [string](#string) |  | The client_secret provided to the node upon successful onboarding |
| project_id | [string](#string) |  | The project_id associated with the node, identifying the project to which the node belongs |





 


<a name="onboardingmgr-OnboardStreamResponse-NodeState"></a>

### OnboardStreamResponse.NodeState
NodeState represents state of the device as stored in Infra Inventory

| Name | Number | Description |
| ---- | ------ | ----------- |
| UNSPECIFIED | 0 | Edge Node state is unspecified or unknown |
| REGISTERED | 1 | Allow to retry, Node is registered but not yet onboarded |
| ONBOARDED | 2 | Edge Node successfully onboarded |


 

 


<a name="onboardingmgr-InteractiveOnboardingService"></a>

### InteractiveOnboardingService
Interactive Onboarding

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| CreateNodes | [NodeRequest](#onboardingmgr-NodeRequest) | [NodeResponse](#onboardingmgr-NodeResponse) |  |


<a name="onboardingmgr-NonInteractiveOnboardingService"></a>

### NonInteractiveOnboardingService
Non Interactive Onboarding

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| OnboardNodeStream | [OnboardStreamRequest](#onboardingmgr-OnboardStreamRequest) stream | [OnboardStreamResponse](#onboardingmgr-OnboardStreamResponse) stream | OnboardNodeStream establishes a bidirectional stream between the Edge Node and the Onboarding Manager It allows Edge Node to send stream requests and receive responses |

 



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


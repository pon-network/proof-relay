### Version

1.0.0

## Content negotiation

### URI Schemes
  * http

### Consumes
  * application/json

### Produces
  * application/json

## All endpoints

###  reporter_api

| Method  | URI     | Name   | Summary |
|---------|---------|--------|---------|
| POST | /blocksubmissions | [builder submission](#builder-submission) | Get Block Submissions Of Builders. |
| POST | /proposerblindedblocks | [proposer blinded block](#proposer-blinded-block) | Get Proposer Payload Delivered. |
| POST | /payloaddelivered | [proposer payload delivered](#proposer-payload-delivered) | Get Proposer Payload Delivered. |
  


## Paths

### <span id="builder-submission"></span> Get Block Submissions Of Builders. (*builderSubmission*)

```
POST /blocksubmissions
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| slot_lower | `query` | uint64 (formatted integer) | `uint64` |  |  |  | Slot Number From Which Needed |
| slot_upper | `query` | uint64 (formatted integer) | `uint64` |  |  |  | Slot Number To Which Needed |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#builder-submission-200) | OK | Blocks Provided Correctly | ✓ | [schema](#builder-submission-200-schema) |
| [204](#builder-submission-204) | No Content | No Builder Submissions | ✓ | [schema](#builder-submission-204-schema) |
| [400](#builder-submission-400) | Bad Request | Invalid Parameter Provided | ✓ | [schema](#builder-submission-400-schema) |
| [500](#builder-submission-500) | Internal Server Error | Server Error | ✓ | [schema](#builder-submission-500-schema) |

#### Responses


##### <span id="builder-submission-200"></span> 200 - Blocks Provided Correctly
Status: OK

###### <span id="builder-submission-200-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| builder_bids | [] | `[]` |  |  | Builder Bids |

##### <span id="builder-submission-204"></span> 204 - No Builder Submissions
Status: No Content

###### <span id="builder-submission-204-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| slot | [] | `[]` |  |  | Empty List Of Bids |

##### <span id="builder-submission-400"></span> 400 - Invalid Parameter Provided
Status: Bad Request

###### <span id="builder-submission-400-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| error | string | `string` |  |  | Error Parameters |

##### <span id="builder-submission-500"></span> 500 - Server Error
Status: Internal Server Error

###### <span id="builder-submission-500-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| error | string | `string` |  |  | Error In The Server |

### <span id="proposer-blinded-block"></span> Get Proposer Payload Delivered. (*proposerBlindedBlock*)

```
POST /proposerblindedblocks
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| slot_lower | `query` | uint64 (formatted integer) | `uint64` |  |  |  | Slot Number From Which Needed |
| slot_upper | `query` | uint64 (formatted integer) | `uint64` |  |  |  | Slot Number To Which Needed |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#proposer-blinded-block-200) | OK | Blocks Provided Correctly | ✓ | [schema](#proposer-blinded-block-200-schema) |
| [204](#proposer-blinded-block-204) | No Content | No Builder Submissions | ✓ | [schema](#proposer-blinded-block-204-schema) |
| [400](#proposer-blinded-block-400) | Bad Request | Invalid Parameter Provided | ✓ | [schema](#proposer-blinded-block-400-schema) |
| [500](#proposer-blinded-block-500) | Internal Server Error | Server Error | ✓ | [schema](#proposer-blinded-block-500-schema) |

#### Responses


##### <span id="proposer-blinded-block-200"></span> 200 - Blocks Provided Correctly
Status: OK

###### <span id="proposer-blinded-block-200-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| builder_blinded_blocks | [] | `[]` |  |  | Blinded Beacon Blocks |

##### <span id="proposer-blinded-block-204"></span> 204 - No Builder Submissions
Status: No Content

###### <span id="proposer-blinded-block-204-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| slot | [] | `[]` |  |  | Empty List Of Bids |

##### <span id="proposer-blinded-block-400"></span> 400 - Invalid Parameter Provided
Status: Bad Request

###### <span id="proposer-blinded-block-400-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| error | string | `string` |  |  | Error Parameters |

##### <span id="proposer-blinded-block-500"></span> 500 - Server Error
Status: Internal Server Error

###### <span id="proposer-blinded-block-500-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| error | string | `string` |  |  | Error In The Server |

### <span id="proposer-payload-delivered"></span> Get Proposer Payload Delivered. (*proposerPayloadDelivered*)

```
POST /payloaddelivered
```

#### Parameters

| Name | Source | Type | Go type | Separator | Required | Default | Description |
|------|--------|------|---------|-----------| :------: |---------|-------------|
| slot_lower | `query` | uint64 (formatted integer) | `uint64` |  |  |  | Slot Number From Which Needed |
| slot_upper | `query` | uint64 (formatted integer) | `uint64` |  |  |  | Slot Number To Which Needed |

#### All responses
| Code | Status | Description | Has headers | Schema |
|------|--------|-------------|:-----------:|--------|
| [200](#proposer-payload-delivered-200) | OK | Blocks Provided Correctly | ✓ | [schema](#proposer-payload-delivered-200-schema) |
| [204](#proposer-payload-delivered-204) | No Content | No Builder Submissions | ✓ | [schema](#proposer-payload-delivered-204-schema) |
| [400](#proposer-payload-delivered-400) | Bad Request | Invalid Parameter Provided | ✓ | [schema](#proposer-payload-delivered-400-schema) |
| [500](#proposer-payload-delivered-500) | Internal Server Error | Server Error | ✓ | [schema](#proposer-payload-delivered-500-schema) |

#### Responses


##### <span id="proposer-payload-delivered-200"></span> 200 - Blocks Provided Correctly
Status: OK

###### <span id="proposer-payload-delivered-200-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| payload_delivered | [] | `[]` |  |  | Delivered Payloads |

##### <span id="proposer-payload-delivered-204"></span> 204 - No Builder Submissions
Status: No Content

###### <span id="proposer-payload-delivered-204-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| slot | [] | `[]` |  |  | Empty List Of Bids |

##### <span id="proposer-payload-delivered-400"></span> 400 - Invalid Parameter Provided
Status: Bad Request

###### <span id="proposer-payload-delivered-400-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| error | string | `string` |  |  | Error Parameters |

##### <span id="proposer-payload-delivered-500"></span> 500 - Server Error
Status: Internal Server Error

###### <span id="proposer-payload-delivered-500-schema"></span> Schema

###### Response headers

| Name | Type | Go type | Separator | Default | Description |
|------|------|---------|-----------|---------|-------------|
| error | string | `string` |  |  | Error In The Server |

## Models

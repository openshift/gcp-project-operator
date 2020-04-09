# API

## ProjectClaim CR

### Metadata

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | ProjectClaim name | string | true |
| namespace | Namespace of ProjectClaim | string | true |

### Spec

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| region | GCP Region Zone | string | true |
| gcpProjectID | GCP Project unique identifier | string | true |

#### gcpCredentialSecret

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | secret name | string | true |
| namespace | secret's namespace | string | true |

#### projectReferenceCRLink

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | projectReference instance name | string | false |
| namespace | projectReference instance namespace | string | false |

#### legalEntity

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| name | customer entity name | string | true |
| id | customer identification number | string | true |

## ProjectReference CR

It is generated and populated by the Operator
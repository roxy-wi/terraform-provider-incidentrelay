---
page_title: "Authentication - IncidentRelay Provider"
subcategory: "Guides"
description: |-
  Authentication patterns for the IncidentRelay Terraform provider.
---

# Authentication

The provider supports two authentication modes:

- Bearer token authentication with `token` or `INCIDENTRELAY_TOKEN`.
- Username/password login against `/api/auth/login`.

Token authentication is recommended for CI, Terraform Cloud, and automation.
Username/password authentication is useful for local development or bootstrap
workflows.

## Token Authentication

```hcl
variable "incidentrelay_token" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = "https://incidentrelay.example.com"
  token    = var.incidentrelay_token
}
```

For CI, prefer environment variables:

```sh
export INCIDENTRELAY_BASE_URL="https://incidentrelay.example.com"
export INCIDENTRELAY_TOKEN="..."
terraform plan
```

## Username And Password Authentication

```hcl
variable "incidentrelay_username" {
  type = string
}

variable "incidentrelay_password" {
  type      = string
  sensitive = true
}

provider "incidentrelay" {
  base_url = var.incidentrelay_base_url
  username = var.incidentrelay_username
  password = var.incidentrelay_password
}
```

The provider exchanges the credentials for an access token during provider
configuration. The password is marked sensitive in Terraform.

## Local Development With Self-Signed TLS

```hcl
provider "incidentrelay" {
  base_url                 = "https://localhost:5000"
  token                    = var.incidentrelay_token
  insecure_skip_tls_verify = true
}
```

Do not use `insecure_skip_tls_verify` in production automation.

## GitHub Actions Example

```yaml
name: Terraform

on:
  pull_request:

jobs:
  plan:
    runs-on: ubuntu-latest
    env:
      INCIDENTRELAY_BASE_URL: ${{ secrets.INCIDENTRELAY_BASE_URL }}
      INCIDENTRELAY_TOKEN: ${{ secrets.INCIDENTRELAY_TOKEN }}
    steps:
      - uses: actions/checkout@v6
      - uses: hashicorp/setup-terraform@v3
      - run: terraform init
      - run: terraform validate
      - run: terraform plan -input=false
```

## Secret Handling

- Keep tokens in the secret store of your CI platform.
- Do not commit `.tfvars` files containing secrets.
- Use sensitive Terraform variables for tokens and passwords.
- Rotate tokens used by CI separately from human user credentials.

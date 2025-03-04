---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "sonarqube_user_tokens Data Source - terraform-provider-sonarqube"
subcategory: ""
description: |-
  Use this data source to get Sonarqube user token resources
---

# sonarqube_user_tokens (Data Source)

Use this data source to get Sonarqube user token resources

## Example Usage

```terraform
data "sonarqube_user_tokens" "user_tokens_admin" {
  login = "admin"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `ignore_missing` (Boolean) If set to true, the data source will not fail if the user does not exist.
- `login_name` (String) Search user tokens for the specified login name. Otherwise, tokens for the current user are listed. This login must exist and be active.

### Read-Only

- `id` (String) The ID of this resource.
- `user_tokens` (List of Object) The list of user tokens. (see [below for nested schema](#nestedatt--user_tokens))

<a id="nestedatt--user_tokens"></a>
### Nested Schema for `user_tokens`

Read-Only:

- `created_at` (String)
- `expiration_date` (String)
- `id` (String)
- `name` (String)
- `project_key` (String)
- `type` (String)

package sonarqube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/satori/uuid"
)

// GetGroupPermissions struct
type GetGroupPermissions struct {
	Paging Paging            `json:"paging"`
	Groups []GroupPermission `json:"groups"`
}

// GroupPermission struct
type GroupPermission struct {
	Id          string   `json:"id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions,omitempty"`
}

// Returns the resource represented by this file.
func resourceSonarqubePermissions() *schema.Resource {
	return &schema.Resource{
		Description: "Provides a Sonarqube Permissions resource. This can be used to manage global and project permissions.",
		Create:      resourceSonarqubePermissionsCreate,
		Read:        resourceSonarqubePermissionsRead,
		Delete:      resourceSonarqubePermissionsDelete,

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"login_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"login_name", "group_name", "special_group_name"},
				Description:  "The name of the user that should get the specified permissions. Changing this forces a new resource to be created. Cannot be used with `group_name` and `special_group_name`.",
			},
			"group_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"login_name", "group_name", "special_group_name"},
				Description:  "The name of the Group that should get the specified permissions. Changing this forces a new resource to be created. Cannot be used with `login_name` and `special_group_name`.",
			},
			"special_group_name": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ExactlyOneOf: []string{"login_name", "group_name", "special_group_name"},
				ValidateDiagFunc: validation.ToDiagFunc(
					validation.StringInSlice(
						[]string{"project_creator"},
						false,
					),
				),
				Description: "The name of the Special Group that should get the specified permissions. Changing this forces a new resource to be created. Cannot be used with `login_name` and `group_name`.",
			},
			"project_key": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"special_group_name", "template_id", "template_name"},
				Description:   "Specify if you want to apply project level permissions. Changing this forces a new resource to be created. Cannot be used with `special_group_name`, `template_id` and `template_name`.",
			},
			"template_id": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"project_key", "template_name"},
				Description:   "Specify if you want to apply the permissions to a permission template. Changing this forces a new resource to be created. Cannot be used with `project_key` and `template_name`.",
			},
			"template_name": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				ConflictsWith: []string{"project_key", "template_id"},
				Description:   "Specify if you want to apply the permissions to a permission template. Changing this forces a new resource to be created. Cannot be used with `project_key` and `template_id`.",
			},
			"permissions": {
				Type:     schema.TypeList,
				MinItems: 1,
				Required: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "A list of permissions that should be applied. Changing this forces a new resource to be created. Possible values are: `admin`, `codeviewer`, `issueadmin`, `securityhotspotadmin`, `scan`, `user`.",
			},
		},
	}
}

func resourceSonarqubePermissionsCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	permissions := expandPermissions(d)

	// build the base query
	RawQuery := url.Values{}

	// if the permissions should be applied to a project
	// we append the project_key to the request
	if projectKey, ok := d.GetOk("project_key"); ok {
		RawQuery.Add("projectKey", projectKey.(string))
	}

	// we use different API endpoints and request params
	// based on the target principal type (group or user or
	// special group) and if its a direct or template permission
	if _, ok := d.GetOk("login_name"); ok {
		// user permission
		RawQuery.Add("login", d.Get("login_name").(string))
		if templateID, ok := d.GetOk("template_id"); ok {
			// template user permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/add_user_to_template"
			RawQuery.Add("templateId", templateID.(string))
			// name provide instead of id
		} else if templateName, ok := d.GetOk("template_name"); ok {
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/add_user_to_template"
			RawQuery.Add("templateName", templateName.(string))
		} else {
			// direct user permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/add_user"
		}
	} else if _, ok := d.GetOk("group_name"); ok {
		// group permission
		RawQuery.Add("groupName", d.Get("group_name").(string))
		if templateID, ok := d.GetOk("template_id"); ok {
			// template group permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/add_group_to_template"
			RawQuery.Add("templateId", templateID.(string))
		} else if templateName, ok := d.GetOk("template_name"); ok {
			// name provide instead of id
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/add_group_to_template"
			RawQuery.Add("templateName", templateName.(string))
		} else {
			// direct user permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/add_group"
		}
	} else {
		// special group permission set to project creator
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/add_project_creator_to_template"
		if templateID, ok := d.GetOk("template_id"); ok {
			// template project creator permission
			RawQuery.Add("templateId", templateID.(string))
		} else if templateName, ok := d.GetOk("template_name"); ok {
			// name provide instead of id
			RawQuery.Add("templateName", templateName.(string))
		} else {
			return fmt.Errorf("resourceSonarqubePermissionsCreate: 'templateId' or 'templateName' must be set when 'special_group_name' is set to 'project_creator'")
		}
	}

	// loop through all permissions that should be applied
	for _, permission := range permissions {
		CurrentRawQuery := RawQuery
		CurrentRawQuery.Del("permission")
		CurrentRawQuery.Add("permission", permission)
		sonarQubeURL.RawQuery = CurrentRawQuery.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL.String(),
			http.StatusNoContent,
			"resourceSonarqubePermissionsCreate",
		)
		if err != nil {
			return fmt.Errorf("error creating Sonarqube permission: %+v", err)
		}
		defer resp.Body.Close()
	}

	// generate a unique ID
	d.SetId(uuid.NewV4().String())
	return resourceSonarqubePermissionsRead(d, m)
}

func resourceSonarqubePermissionsRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL

	// build the base query
	RawQuery := url.Values{
		// set the page size to 100
		"ps": []string{"100"},
	}

	// if the permissions should be applied to a project
	// we append the project_key to the request
	if projectKey, ok := d.GetOk("project_key"); ok {
		RawQuery.Add("projectKey", projectKey.(string))
	}

	// we use different API endpoints and request params
	// based on the target principal type (group or user or
	// special group) and if its a direct or template permission
	if _, ok := d.GetOk("login_name"); ok {
		// permission target is USER
		if templateID, ok := d.GetOk("template_id"); ok {
			// template user permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/template_users"
			RawQuery.Add("templateId", templateID.(string))
			// name provide instead of id
		} else if templateName, ok := d.GetOk("template_name"); ok {
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/template_users"
			RawQuery.Add("templateName", templateName.(string))
		} else {
			// direct user permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/users"
		}
		sonarQubeURL.RawQuery = RawQuery.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"GET",
			sonarQubeURL.String(),
			http.StatusOK,
			"resourceSonarqubePermissionsRead",
		)
		if err != nil {
			return fmt.Errorf("error reading Sonarqube permissions: %+v", err)
		}
		defer resp.Body.Close()

		// Decode response into struct
		users := GetUser{}
		err = json.NewDecoder(resp.Body).Decode(&users)
		if err != nil {
			return fmt.Errorf("resourceSonarqubePermissionsRead: Failed to decode json into struct: %+v", err)
		}

		// Loop over all groups to see if the group we need exists.
		loginName := d.Get("login_name").(string)
		for _, value := range users.Users {
			if strings.EqualFold(value.Login, loginName) {
				d.Set("login_name", value.Login)
				d.Set("permissions", flattenPermissions(&value.Permissions))
				return nil
			}
		}
	} else if _, ok := d.GetOk("group_name"); ok {
		// permission target is GROUP
		if templateID, ok := d.GetOk("template_id"); ok {
			// template group permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/template_groups"
			RawQuery.Add("templateId", templateID.(string))
		} else if templateName, ok := d.GetOk("template_name"); ok {
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/template_groups"
			RawQuery.Add("templateName", templateName.(string))
		} else {
			// direct group permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/groups"
		}
		sonarQubeURL.RawQuery = RawQuery.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"GET",
			sonarQubeURL.String(),
			http.StatusOK,
			"resourceSonarqubePermissionsRead",
		)
		if err != nil {
			return fmt.Errorf("error reading Sonarqube permissions: %+v", err)
		}
		defer resp.Body.Close()

		// Decode response into struct
		groups := GetGroupPermissions{}
		err = json.NewDecoder(resp.Body).Decode(&groups)
		if err != nil {
			return fmt.Errorf("resourceSonarqubePermissionsRead: Failed to decode json into struct: %+v", err)
		}

		// Loop over all groups to see if the group we need exists.
		groupName := d.Get("group_name").(string)
		for _, value := range groups.Groups {
			if strings.EqualFold(value.Name, groupName) {
				d.Set("group_name", value.Name)
				d.Set("permissions", flattenPermissions(&value.Permissions))
				return nil
			}
		}
	} else {
		// permission target is PROJECT CREATOR set to project creator
		if templateName, ok := d.GetOk("template_name"); ok {
			RawQuery.Add("templateName", templateName.(string))
		}
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/search_templates"
		sonarQubeURL.RawQuery = RawQuery.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"GET",
			sonarQubeURL.String(),
			http.StatusOK,
			"resourceSonarqubePermissionsRead",
		)
		if err != nil {
			return fmt.Errorf("error reading Sonarqube permissions: %+v", err)
		}
		defer resp.Body.Close()

		// Decode response into struct
		permissionTemplates := GetPermissionTemplates{}
		err = json.NewDecoder(resp.Body).Decode(&permissionTemplates)
		if err != nil {
			return fmt.Errorf("resourceSonarqubePermissionsRead: Failed to decode json into struct: %+v", err)
		}

		// Loop over all permission templates
		templateId := d.Get("template_id").(string)
		templateName := d.Get("template_name").(string)
		for _, value := range permissionTemplates.PermissionTemplates {
			if strings.EqualFold(value.ID, templateId) || strings.EqualFold(value.Name, templateName) {
				d.Set("special_group_name", "project_creator")
				d.Set("permissions", flattenProjectCreatorPermissions(&value.Permissions))
				return nil
			}
		}
	}

	return fmt.Errorf("resourceSonarqubePermissionsRead: Unable to find group permissions for group: %+v", d.Id())
}

func resourceSonarqubePermissionsDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	permissions := expandPermissions(d)

	// build the base query
	RawQuery := url.Values{}

	// if the permissions should be applied to a project
	// we append the project_key to the request
	if projectKey, ok := d.GetOk("project_key"); ok {
		RawQuery.Add("projectKey", projectKey.(string))
	}

	// we use different API endpoints and request params
	// based on the target principal type (group or user
	// or special group)
	if _, ok := d.GetOk("login_name"); ok {
		// permission target is USER
		if templateID, ok := d.GetOk("template_id"); ok {
			// template user permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/remove_user_from_template"
			RawQuery.Add("templateId", templateID.(string))
		} else if templateName, ok := d.GetOk("template_name"); ok {
			// name provide instead of id
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/remove_user_from_template"
			RawQuery.Add("templateName", templateName.(string))
		} else {
			// direct user permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/remove_user"
		}
		RawQuery.Add("login", d.Get("login_name").(string))
		sonarQubeURL.RawQuery = RawQuery.Encode()
	} else if _, ok := d.GetOk("group_name"); ok {
		// permission target is GROUP
		if templateID, ok := d.GetOk("template_id"); ok {
			// template group permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/remove_group_from_template"
			RawQuery.Add("templateId", templateID.(string))
		} else if templateName, ok := d.GetOk("template_name"); ok {
			// name provide instead of id
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/remove_group_from_template"
			RawQuery.Add("templateName", templateName.(string))
		} else {
			// direct group permission
			sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/remove_group"
		}
		RawQuery.Add("groupName", d.Get("group_name").(string))
		sonarQubeURL.RawQuery = RawQuery.Encode()
	} else {
		// permission target is SPECIAL GROUP set to project creator
		sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/permissions/remove_project_creator_from_template"
		if templateID, ok := d.GetOk("template_id"); ok {
			// template project creator permission
			RawQuery.Add("templateId", templateID.(string))
		} else if templateName, ok := d.GetOk("template_name"); ok {
			// name provide instead of id
			RawQuery.Add("templateName", templateName.(string))
		} else {
			return fmt.Errorf("resourceSonarqubePermissionsDelete: 'templateId' or 'templateName' must be set when 'special_group_name' is set to 'project_creator'")
		}
	}

	// loop through all permissions that should be applied
	for _, permission := range permissions {
		CurrentRawQuery := RawQuery
		CurrentRawQuery.Del("permission")
		CurrentRawQuery.Add("permission", permission)
		sonarQubeURL.RawQuery = CurrentRawQuery.Encode()

		resp, err := httpRequestHelper(
			m.(*ProviderConfiguration).httpClient,
			"POST",
			sonarQubeURL.String(),
			http.StatusNoContent,
			"resourceSonarqubePermissionsDelete",
		)
		if err != nil {
			return fmt.Errorf("error creating Sonarqube permission: %+v", err)
		}
		defer resp.Body.Close()
	}

	return nil
}

func expandPermissions(d *schema.ResourceData) []string {
	expandedPermissions := make([]string, 0)
	flatPermissions := d.Get("permissions").([]interface{})
	for _, permission := range flatPermissions {
		expandedPermissions = append(expandedPermissions, permission.(string))
	}

	return expandedPermissions
}

func flattenPermissions(input *[]string) []interface{} {
	flatPermissions := make([]interface{}, 0)
	if input == nil {
		return flatPermissions
	}

	for _, permission := range *input {
		flatPermissions = append(flatPermissions, permission)
	}

	return flatPermissions
}

func flattenProjectCreatorPermissions(input *[]PermissionTemplatePermission) []interface{} {
	flatPermissions := make([]interface{}, 0)
	if input == nil {
		return flatPermissions
	}

	for _, permission := range *input {
		if permission.WithProjectCreator {
			flatPermissions = append(flatPermissions, permission.Key)
		}
	}

	return flatPermissions
}

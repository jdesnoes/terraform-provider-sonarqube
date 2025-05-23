package sonarqube

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// GetAlmGitlab for unmarshalling response body from alm list definitions. With only gitlab populated
type GetAlmGitlab struct {
	Gitlab []struct {
		Key                 string `json:"key"`
		URL                 string `json:"url"`
		PersonalAccessToken string `json:"personalAccessToken,omitempty"`
	} `json:"gitlab"`
}

// Returns the resource represented by this file.
func resourceSonarqubeAlmGitlab() *schema.Resource {
	return &schema.Resource{
		Description: `Provides a Sonarqube GitLab Alm/Devops Platform Integration resource. This can be used to create and manage a Alm/Devops
Platform Integration for GitLab.`,
		Create: resourceSonarqubeAlmGitlabCreate,
		Read:   resourceSonarqubeAlmGitlabRead,
		Update: resourceSonarqubeAlmGitlabUpdate,
		Delete: resourceSonarqubeAlmGitlabDelete,

		// Define the fields of this schema.
		Schema: map[string]*schema.Schema{
			"key": {
				Type:             schema.TypeString,
				Required:         true,
				ForceNew:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 200)),
				Description:      "Unique key of the GitLab instance setting. Maximum length: 200",
			},
			"personal_access_token": {
				Type:             schema.TypeString,
				Required:         true,
				Sensitive:        true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 2000)),
				Description:      "GitLab App personal access token with the `read_api` scope. See [this doc](https://docs.sonarqube.org/latest/devops-platform-integration/gitlab-integration/#importing-your-gitlab-projects-into-sonarqube) for more information. Maximum length: 2000",
			},
			"url": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringLenBetween(1, 2000)),
				Description:      "GitLab API URL. Maximum length: 2000",
			},
		},
	}
}

func resourceSonarqubeAlmGitlabCreate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/create_gitlab"

	sonarQubeURL.RawQuery = url.Values{
		"key":                 []string{d.Get("key").(string)},
		"personalAccessToken": []string{d.Get("personal_access_token").(string)},
		"url":                 []string{d.Get("url").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeAlmGitlabCreate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d.SetId(d.Get("key").(string))

	return resourceSonarqubeAlmGitlabRead(d, m)
}

func resourceSonarqubeAlmGitlabRead(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/list_definitions"

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"GET",
		sonarQubeURL.String(),
		http.StatusOK,
		"resourceSonarqubeAlmGitlabRead",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Decode response into struct
	AlmGitlabReadResponse := GetAlmGitlab{}
	err = json.NewDecoder(resp.Body).Decode(&AlmGitlabReadResponse)
	if err != nil {
		return fmt.Errorf("resourceSonarqubeAlmGitlabRead: Failed to decode json into struct: %+v", err)
	}
	// Loop over all GitHub instances to see if the Alm instance exists.
	for _, value := range AlmGitlabReadResponse.Gitlab {
		if d.Id() == value.Key {
			errKey := d.Set("key", value.Key)
			errUrl := d.Set("url", value.URL)
			// The personal_access_token is a secured property that is not returned
			// d.Set("personal_access_token", value.PersonalAccessToken)
			return errors.Join(errKey, errUrl)
		}
	}
	return fmt.Errorf("resourceSonarqubeGitlabBindingRead: Failed to find gitlab binding: %+v", d.Id())
}

func resourceSonarqubeAlmGitlabUpdate(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/update_gitlab"
	sonarQubeURL.RawQuery = url.Values{
		"key":                 []string{d.Id()},
		"newKey":              []string{d.Get("key").(string)},
		"personalAccessToken": []string{d.Get("personal_access_token").(string)},
		"url":                 []string{d.Get("url").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeAlmGitlabUpdate",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return resourceSonarqubeAlmGitlabRead(d, m)
}

func resourceSonarqubeAlmGitlabDelete(d *schema.ResourceData, m interface{}) error {
	sonarQubeURL := m.(*ProviderConfiguration).sonarQubeURL
	sonarQubeURL.Path = strings.TrimSuffix(sonarQubeURL.Path, "/") + "/api/alm_settings/delete"
	sonarQubeURL.RawQuery = url.Values{
		"key": []string{d.Get("key").(string)},
	}.Encode()

	resp, err := httpRequestHelper(
		m.(*ProviderConfiguration).httpClient,
		"POST",
		sonarQubeURL.String(),
		http.StatusNoContent,
		"resourceSonarqubeAlmGitlabDelete",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

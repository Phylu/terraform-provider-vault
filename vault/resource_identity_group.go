package vault

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/vault/api"
)

const identityGroupPath = "/identity/group"

func identityGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: identityGroupCreate,
		Update: identityGroupUpdate,
		Read:   identityGroupRead,
		Delete: identityGroupDelete,
		Exists: identityGroupExists,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the group.",
				ForceNew:    true,
			},

			"type": {
				Type:        schema.TypeString,
				Description: "Type of the group, internal or external. Defaults to internal.",
				ForceNew:    true,
				Optional:    true,
				Default:     "internal",
			},

			"metadata": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "Metadata to be associated with the group.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"policies": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Policies to be tied to the group.",
			},

			"member_group_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Group IDs to be assigned as group members.",
			},

			"member_entity_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Entity IDs to be assigned as group members.",
			},

			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the group.",
			},
		},
	}
}

func identityGroupUpdateFields(d *schema.ResourceData, data map[string]interface{}) {
	if policies, ok := d.GetOk("policies"); ok {
		data["policies"] = policies
	}

	if memberEntityIDs, ok := d.GetOk("member_entity_ids"); ok {
		data["member_entity_ids"] = memberEntityIDs
	}

	if memberGroupIDs, ok := d.GetOk("member_group_ids"); ok {
		data["member_group_ids"] = memberGroupIDs
	}

	if metadata, ok := d.GetOk("metadata"); ok {
		data["metadata"] = metadata
	}
}

func identityGroupCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)

	name := d.Get("name").(string)
	typeValue := d.Get("type").(string)

	path := identityGroupPath

	data := map[string]interface{}{
		"name": name,
		"type": typeValue,
	}

	identityGroupUpdateFields(d, data)

	resp, err := client.Logical().Write(path, data)

	if err != nil {
		return fmt.Errorf("error writing IdentityGroup to %q: %s", name, err)
	}
	log.Printf("[DEBUG] Wrote IdentityGroup %q", name)

	d.Set("id", resp.Data["id"])

	d.SetId(resp.Data["id"].(string))

	return identityGroupRead(d, meta)
}

func identityGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	id := d.Id()

	log.Printf("[DEBUG] Updating IdentityGroup %q", id)
	path := identityGroupIDPath(id)

	data := map[string]interface{}{}

	identityGroupUpdateFields(d, data)

	_, err := client.Logical().Write(path, data)

	if err != nil {
		return fmt.Errorf("error updating IdentityGroup %q: %s", id, err)
	}
	log.Printf("[DEBUG] Updated IdentityGroup %q", id)

	return identityGroupRead(d, meta)
}

func identityGroupRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	id := d.Id()

	path := identityGroupIDPath(id)

	log.Printf("[DEBUG] Reading IdentityGroup %s from %q", id, path)
	resp, err := client.Logical().Read(path)
	if err != nil {
		// We need to check if the secret_id has expired
		if isExpiredTokenErr(err) {
			return nil
		}
		return fmt.Errorf("error reading AppRole auth backend role SecretID %q: %s", id, err)
	}
	log.Printf("[DEBUG] Read IdentityGroup %s", id)
	if resp == nil {
		log.Printf("[WARN] IdentityGroup %q not found, removing from state", id)
		d.SetId("")
		return nil
	}

	for _, k := range []string{"name", "type", "metadata", "member_entity_ids", "member_group_ids"} {
		d.Set(k, resp.Data[k])
	}
	return nil
}

func identityGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*api.Client)
	id := d.Id()

	path := identityGroupIDPath(id)

	log.Printf("[DEBUG] Deleting IdentityGroup %q", id)
	_, err := client.Logical().Delete(path)
	if err != nil {
		return fmt.Errorf("error IdentityGroup %q", id)
	}
	log.Printf("[DEBUG] Deleted IdentityGroup %q", id)

	return nil
}

func identityGroupExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*api.Client)
	id := d.Id()

	path := identityGroupIDPath(id)
	key := id

	// use the name if no ID is set
	if len(id) == 0 {
		key = d.Get("name").(string)
		path = identityGroupNamePath(key)
	}

	log.Printf("[DEBUG] Checking if IdentityGroup %q exists", key)
	resp, err := client.Logical().Read(path)
	if err != nil {
		return true, fmt.Errorf("error checking if IdentityGroup %q exists: %s", key, err)
	}
	log.Printf("[DEBUG] Checked if IdentityGroup %q exists", key)

	return resp != nil, nil
}

func identityGroupNamePath(name string) string {
	return fmt.Sprintf("%s/name/%s", identityGroupPath, name)
}

func identityGroupIDPath(id string) string {
	return fmt.Sprintf("%s/id/%s", identityGroupPath, id)
}

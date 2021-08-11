package securityhub

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/securityhub"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-aws/internal/client"
)

func ResourceOrganizationConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSecurityHubOrganizationConfigurationUpdate,
		Read:   resourceAwsSecurityHubOrganizationConfigurationRead,
		Update: resourceAwsSecurityHubOrganizationConfigurationUpdate,
		Delete: schema.Noop,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"auto_enable": {
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

func resourceAwsSecurityHubOrganizationConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).SecurityHubConn

	input := &securityhub.UpdateOrganizationConfigurationInput{
		AutoEnable: aws.Bool(d.Get("auto_enable").(bool)),
	}

	_, err := conn.UpdateOrganizationConfiguration(input)

	if err != nil {
		return fmt.Errorf("error updating Security Hub Organization Configuration (%s): %w", d.Id(), err)
	}

	d.SetId(meta.(*client.AWSClient).AccountID)

	return resourceAwsSecurityHubOrganizationConfigurationRead(d, meta)
}

func resourceAwsSecurityHubOrganizationConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).SecurityHubConn

	output, err := conn.DescribeOrganizationConfiguration(&securityhub.DescribeOrganizationConfigurationInput{})

	if err != nil {
		return fmt.Errorf("error reading Security Hub Organization Configuration: %w", err)
	}

	d.Set("auto_enable", output.AutoEnable)

	return nil
}
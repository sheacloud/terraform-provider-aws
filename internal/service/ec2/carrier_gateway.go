package ec2

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/terraform-providers/terraform-provider-aws/internal/client"
	"github.com/terraform-providers/terraform-provider-aws/internal/keyvaluetags"
	"github.com/terraform-providers/terraform-provider-aws/internal/tags"
)

func ResourceCarrierGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsEc2CarrierGatewayCreate,
		Read:   resourceAwsEc2CarrierGatewayRead,
		Update: resourceAwsEc2CarrierGatewayUpdate,
		Delete: resourceAwsEc2CarrierGatewayDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		CustomizeDiff: tags.SetTagsDiff,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags":     tags.TagsSchema(),
			"tags_all": tags.TagsSchemaComputed(),

			"vpc_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsEc2CarrierGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).EC2Conn
	defaultTagsConfig := meta.(*client.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(keyvaluetags.New(d.Get("tags").(map[string]interface{})))

	input := &ec2.CreateCarrierGatewayInput{
		TagSpecifications: ec2TagSpecificationsFromKeyValueTags(tags, "carrier-gateway"),
		VpcId:             aws.String(d.Get("vpc_id").(string)),
	}

	log.Printf("[DEBUG] Creating EC2 Carrier Gateway: %s", input)
	output, err := conn.CreateCarrierGateway(input)

	if err != nil {
		return fmt.Errorf("error creating EC2 Carrier Gateway: %w", err)
	}

	d.SetId(aws.StringValue(output.CarrierGateway.CarrierGatewayId))

	_, err = waitCarrierGatewayAvailable(conn, d.Id())

	if err != nil {
		return fmt.Errorf("error waiting for EC2 Carrier Gateway (%s) to become available: %w", d.Id(), err)
	}

	return resourceAwsEc2CarrierGatewayRead(d, meta)
}

func resourceAwsEc2CarrierGatewayRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).EC2Conn
	defaultTagsConfig := meta.(*client.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*client.AWSClient).IgnoreTagsConfig

	carrierGateway, err := findCarrierGatewayByID(conn, d.Id())

	if tfawserr.ErrCodeEquals(err, errCodeInvalidCarrierGatewayIDNotFound) {
		log.Printf("[WARN] EC2 Carrier Gateway (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading EC2 Carrier Gateway (%s): %w", d.Id(), err)
	}

	if carrierGateway == nil || aws.StringValue(carrierGateway.State) == ec2.CarrierGatewayStateDeleted {
		log.Printf("[WARN] EC2 Carrier Gateway (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	arn := arn.ARN{
		Partition: meta.(*client.AWSClient).Partition,
		Service:   ec2.ServiceName,
		Region:    meta.(*client.AWSClient).Region,
		AccountID: aws.StringValue(carrierGateway.OwnerId),
		Resource:  fmt.Sprintf("carrier-gateway/%s", d.Id()),
	}.String()
	d.Set("arn", arn)
	d.Set("owner_id", carrierGateway.OwnerId)
	d.Set("vpc_id", carrierGateway.VpcId)

	tags := keyvaluetags.Ec2KeyValueTags(carrierGateway.Tags).IgnoreAws().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: %w", err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return fmt.Errorf("error setting tags_all: %w", err)
	}

	return nil
}

func resourceAwsEc2CarrierGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).EC2Conn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := keyvaluetags.Ec2UpdateTags(conn, d.Id(), o, n); err != nil {
			return fmt.Errorf("error updating EC2 Carrier Gateway (%s) tags: %w", d.Id(), err)
		}
	}

	return resourceAwsEc2CarrierGatewayRead(d, meta)
}

func resourceAwsEc2CarrierGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).EC2Conn

	log.Printf("[INFO] Deleting EC2 Carrier Gateway (%s)", d.Id())
	_, err := conn.DeleteCarrierGateway(&ec2.DeleteCarrierGatewayInput{
		CarrierGatewayId: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, errCodeInvalidCarrierGatewayIDNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting EC2 Carrier Gateway (%s): %w", d.Id(), err)
	}

	_, err = waitCarrierGatewayDeleted(conn, d.Id())

	if err != nil {
		return fmt.Errorf("error waiting for EC2 Carrier Gateway (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}
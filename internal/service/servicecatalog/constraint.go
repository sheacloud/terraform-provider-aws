package servicecatalog

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/servicecatalog"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/internal/client"
	tfiam "github.com/terraform-providers/terraform-provider-aws/internal/service/iam"
	"github.com/terraform-providers/terraform-provider-aws/internal/tfresource"
	"github.com/terraform-providers/terraform-provider-aws/internal/verify"
)

func ResourceConstraint() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsServiceCatalogConstraintCreate,
		Read:   resourceAwsServiceCatalogConstraintRead,
		Update: resourceAwsServiceCatalogConstraintUpdate,
		Delete: resourceAwsServiceCatalogConstraintDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"accept_language": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      acceptLanguageEnglish,
				ValidateFunc: validation.StringInSlice(acceptLanguage_Values(), false),
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"owner": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"parameters": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateFunc:     validation.StringIsJSON,
				DiffSuppressFunc: verify.SuppressEquivalentJSONDiffs,
			},
			"portfolio_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"product_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice(constraintType_Values(), false),
			},
		},
	}
}

func resourceAwsServiceCatalogConstraintCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).ServiceCatalogConn

	input := &servicecatalog.CreateConstraintInput{
		IdempotencyToken: aws.String(resource.UniqueId()),
		Parameters:       aws.String(d.Get("parameters").(string)),
		PortfolioId:      aws.String(d.Get("portfolio_id").(string)),
		ProductId:        aws.String(d.Get("product_id").(string)),
		Type:             aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("accept_language"); ok {
		input.AcceptLanguage = aws.String(v.(string))
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	var output *servicecatalog.CreateConstraintOutput
	err := resource.Retry(tfiam.PropagationTimeout, func() *resource.RetryError {
		var err error

		output, err = conn.CreateConstraint(input)

		if tfawserr.ErrMessageContains(err, servicecatalog.ErrCodeInvalidParametersException, "profile does not exist") {
			return resource.RetryableError(err)
		}

		if tfawserr.ErrCodeEquals(err, servicecatalog.ErrCodeResourceNotFoundException) {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		output, err = conn.CreateConstraint(input)
	}

	if err != nil {
		return fmt.Errorf("error creating Service Catalog Constraint: %w", err)
	}

	if output == nil || output.ConstraintDetail == nil {
		return fmt.Errorf("error creating Service Catalog Constraint: empty response")
	}

	d.SetId(aws.StringValue(output.ConstraintDetail.ConstraintId))

	return resourceAwsServiceCatalogConstraintRead(d, meta)
}

func resourceAwsServiceCatalogConstraintRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).ServiceCatalogConn

	output, err := waitConstraintReady(conn, d.Get("accept_language").(string), d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] Service Catalog Constraint (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error describing Service Catalog Constraint (%s): %w", d.Id(), err)
	}

	if output == nil || output.ConstraintDetail == nil {
		return fmt.Errorf("error getting Service Catalog Constraint (%s): empty response", d.Id())
	}

	acceptLanguage := d.Get("accept_language").(string)

	if acceptLanguage == "" {
		acceptLanguage = acceptLanguageEnglish
	}

	d.Set("accept_language", acceptLanguage)

	d.Set("parameters", output.ConstraintParameters)
	d.Set("status", output.Status)

	detail := output.ConstraintDetail

	d.Set("description", detail.Description)
	d.Set("owner", detail.Owner)
	d.Set("portfolio_id", detail.PortfolioId)
	d.Set("product_id", detail.ProductId)
	d.Set("type", detail.Type)

	return nil
}

func resourceAwsServiceCatalogConstraintUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).ServiceCatalogConn

	input := &servicecatalog.UpdateConstraintInput{
		Id: aws.String(d.Id()),
	}

	if d.HasChange("accept_language") {
		input.AcceptLanguage = aws.String(d.Get("accept_language").(string))
	}

	if d.HasChange("description") {
		input.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("parameters") {
		input.Parameters = aws.String(d.Get("parameters").(string))
	}

	err := resource.Retry(tfiam.PropagationTimeout, func() *resource.RetryError {
		_, err := conn.UpdateConstraint(input)

		if tfawserr.ErrMessageContains(err, servicecatalog.ErrCodeInvalidParametersException, "profile does not exist") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if tfresource.TimedOut(err) {
		_, err = conn.UpdateConstraint(input)
	}

	if err != nil {
		return fmt.Errorf("error updating Service Catalog Constraint (%s): %w", d.Id(), err)
	}

	return resourceAwsServiceCatalogConstraintRead(d, meta)
}

func resourceAwsServiceCatalogConstraintDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.AWSClient).ServiceCatalogConn

	input := &servicecatalog.DeleteConstraintInput{
		Id: aws.String(d.Id()),
	}

	if v, ok := d.GetOk("accept_language"); ok {
		input.AcceptLanguage = aws.String(v.(string))
	}

	_, err := conn.DeleteConstraint(input)

	if tfawserr.ErrCodeEquals(err, servicecatalog.ErrCodeResourceNotFoundException) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting Service Catalog Constraint (%s): %w", d.Id(), err)
	}

	err = waitConstraintDeleted(conn, d.Get("accept_language").(string), d.Id())

	if err != nil && !tfresource.NotFound(err) {
		return fmt.Errorf("error waiting for Service Catalog Constraint (%s) to be deleted: %w", d.Id(), err)
	}

	return nil
}
package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53resolver"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/terraform-providers/terraform-provider-aws/aws/internal/keyvaluetags"
)

func dataSourceAwsRoute53ResolverRule() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRoute53ResolverRuleRead,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"domain_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ValidateFunc:  validation.StringLenBetween(1, 256),
				ConflictsWith: []string{"resolver_rule_id"},
			},

			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ValidateFunc:  validateRoute53ResolverName,
				ConflictsWith: []string{"resolver_rule_id"},
			},

			"owner_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"resolver_endpoint_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"resolver_rule_id"},
			},

			"resolver_rule_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"domain_name", "name", "resolver_endpoint_id", "rule_type"},
			},

			"rule_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					route53resolver.RuleTypeOptionForward,
					route53resolver.RuleTypeOptionSystem,
					route53resolver.RuleTypeOptionRecursive,
				}, false),
				ConflictsWith: []string{"resolver_rule_id"},
			},

			"share_status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsRoute53ResolverRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).route53resolverconn
	ignoreTagsConfig := meta.(*AWSClient).IgnoreTagsConfig

	var rule *route53resolver.ResolverRule
	if v, ok := d.GetOk("resolver_rule_id"); ok {
		ruleRaw, state, err := route53ResolverRuleRefresh(conn, v.(string))()
		if err != nil {
			return fmt.Errorf("error getting Route53 Resolver rule (%s): %w", v, err)
		}

		if state == route53ResolverRuleStatusDeleted {
			return fmt.Errorf("no Route53 Resolver rules matched found with the id (%q)", v)
		}

		rule = ruleRaw.(*route53resolver.ResolverRule)
	} else {
		req := &route53resolver.ListResolverRulesInput{
			Filters: buildRoute53ResolverAttributeFilterList(map[string]string{
				"DOMAIN_NAME":          d.Get("domain_name").(string),
				"NAME":                 d.Get("name").(string),
				"RESOLVER_ENDPOINT_ID": d.Get("resolver_endpoint_id").(string),
				"TYPE":                 d.Get("rule_type").(string),
			}),
		}

		rules := []*route53resolver.ResolverRule{}
		log.Printf("[DEBUG] Listing Route53 Resolver rules: %s", req)
		err := conn.ListResolverRulesPages(req, func(page *route53resolver.ListResolverRulesOutput, lastPage bool) bool {
			rules = append(rules, page.ResolverRules...)
			return !lastPage
		})
		if err != nil {
			return fmt.Errorf("error getting Route53 Resolver rule: %w", err)
		}
		if n := len(rules); n == 0 {
			return fmt.Errorf("no Route53 Resolver rules matched")
		} else if n > 1 {
			return fmt.Errorf("%d Route53 Resolver rules matched; use additional constraints to reduce matches to a rule", n)
		}

		rule = rules[0]
	}

	d.SetId(aws.StringValue(rule.Id))
	d.Set("arn", rule.Arn)
	// To be consistent with other AWS services that do not accept a trailing period,
	// we remove the suffix from the Domain Name returned from the API
	d.Set("domain_name", trimTrailingPeriod(aws.StringValue(rule.DomainName)))
	d.Set("name", rule.Name)
	d.Set("owner_id", rule.OwnerId)
	d.Set("resolver_endpoint_id", rule.ResolverEndpointId)
	d.Set("resolver_rule_id", rule.Id)
	d.Set("rule_type", rule.RuleType)
	shareStatus := aws.StringValue(rule.ShareStatus)
	d.Set("share_status", shareStatus)
	// https://github.com/hashicorp/terraform-provider-aws/issues/10211
	if shareStatus != route53resolver.ShareStatusSharedWithMe {
		arn := aws.StringValue(rule.Arn)
		tags, err := keyvaluetags.Route53resolverListTags(conn, arn)

		if err != nil {
			return fmt.Errorf("error listing tags for Route 53 Resolver rule (%s): %w", arn, err)
		}

		if err := d.Set("tags", tags.IgnoreAws().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
			return fmt.Errorf("error setting tags: %w", err)
		}
	}

	return nil
}

func buildRoute53ResolverAttributeFilterList(attrs map[string]string) []*route53resolver.Filter {
	filters := []*route53resolver.Filter{}

	for k, v := range attrs {
		if v == "" {
			continue
		}

		filters = append(filters, &route53resolver.Filter{
			Name:   aws.String(k),
			Values: aws.StringSlice([]string{v}),
		})
	}

	return filters
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbv2Types "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

const nlbSecurityGroupRuleKey = "kubernetes.io/rule/nlb/client"

func main() {
	securityGroupId := flag.String("group-id", "", "security group id")
	flag.Parse()

	if *securityGroupId == "" {
		log.Fatal("-group-id is required")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	ec2Client := ec2.NewFromConfig(cfg)
	elbV2Client := elbv2.NewFromConfig(cfg)

	elbOutput, err := elbV2Client.DescribeLoadBalancers(context.Background(), &elbv2.DescribeLoadBalancersInput{})
	if err != nil {
		log.Fatal(err)
	}

	expectedNLBs := map[string]int{}
	for _, nlb := range elbOutput.LoadBalancers {
		if nlb.Type == elbv2Types.LoadBalancerTypeEnumNetwork {
			expectedNLBs[fmt.Sprintf("%s=%s", nlbSecurityGroupRuleKey, *nlb.LoadBalancerName)] = 0
		}
	}

	output, err := ec2Client.DescribeSecurityGroupRules(context.Background(), &ec2.DescribeSecurityGroupRulesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("group-id"),
				Values: []string{*securityGroupId},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, rule := range output.SecurityGroupRules {
		if rule.Description != nil {
			if strings.HasPrefix(*rule.Description, nlbSecurityGroupRuleKey) {
				if _, ok := expectedNLBs[*rule.Description]; ok {
					expectedNLBs[*rule.Description] += 1
				} else {
					fmt.Printf("didn't find %s\n", *rule.Description)
				}
			}
		}
	}

	fmt.Printf("There are %d NLBs\n", len(expectedNLBs))
	fmt.Println(expectedNLBs)
}

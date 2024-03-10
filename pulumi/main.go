package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/cloudwatch"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/autoscaling"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/rds"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/route53"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/secretsmanager"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/sns"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/storage"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"strings"
)

var (
	region  = flag.String("region", "us-west-2", "region")
	vpcCIDR = flag.String("vpc_cidr", "10.1", "vpc-cidr")
)

func getVPCCidr() string {
	return *vpcCIDR + ".0.0/16"
}

func getPrivateCIDR() string {
	return *vpcCIDR + ".2.0/24," + *vpcCIDR + ".4.0/24," + *vpcCIDR + ".6.0/24"
}

func getPublicCIDR() string {
	return *vpcCIDR + ".1.0/24," + *vpcCIDR + ".3.0/24," + *vpcCIDR + ".5.0/24"
}

func getPrivateZones() string {
	return *region + "a," + *region + "b," + *region + "c"
}

func getPublicZones() string {
	return *region + "a," + *region + "b," + *region + "c"
}

func scriptBase64(script string) pulumi.StringPtrInput {
	return pulumi.String(base64.StdEncoding.EncodeToString([]byte(script)))
}

func main() {
	flag.Parse()

	pulumi.Run(func(ctx *pulumi.Context) error {
		vpcCIDR := getVPCCidr()
		//VPC
		myVpc, err := ec2.NewVpc(ctx, "pulumi-vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String(vpcCIDR),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("pulumi-vpc"),
			},
		})
		if err != nil {
			return err
		}

		//public subnets
		publicSubnetZones := strings.Split(getPublicZones(), ",")
		publicSubnetIPs := strings.Split(getPublicCIDR(), ",")

		publicSubnets := make(map[string]interface{})
		publicSubnets["subnet_ids"] = make([]interface{}, 0, len(publicSubnetZones))

		for idx, availabilityZone := range publicSubnetZones {
			subnetArgs := &ec2.SubnetArgs{
				VpcId:            myVpc.ID(),
				CidrBlock:        pulumi.String(publicSubnetIPs[idx]),
				AvailabilityZone: pulumi.String(availabilityZone),
				Tags: pulumi.StringMap{
					"zone": pulumi.String(publicSubnetZones[idx]),
					"ip":   pulumi.String(publicSubnetIPs[idx]),
				},
			}

			publicSubnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-public-subnet-%d", "csye", idx), subnetArgs)
			if err != nil {
				return err
			}

			publicSubnets["subnet_ids"] = append(publicSubnets["subnet_ids"].([]interface{}), publicSubnet.ID())
		}

		//private subnets
		privateSubnetZones := strings.Split(getPrivateZones(), ",")
		privateSubnetIPs := strings.Split(getPrivateCIDR(), ",")

		privateSubnets := make(map[string]interface{})
		privateSubnets["subnet_ids"] = make([]interface{}, 0, len(privateSubnetZones))

		for idx, availabilityZone := range privateSubnetZones {
			subnetArgs := &ec2.SubnetArgs{
				VpcId:            myVpc.ID(),
				CidrBlock:        pulumi.String(privateSubnetIPs[idx]),
				AvailabilityZone: pulumi.String(availabilityZone),
				Tags: pulumi.StringMap{
					"zone": pulumi.String(privateSubnetZones[idx]),
					"ip":   pulumi.String(privateSubnetIPs[idx]),
				},
			}

			privateSubnet, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-private-subnet-%d", "csye", idx), subnetArgs)
			if err != nil {
				return err
			}

			privateSubnets["subnet_ids"] = append(privateSubnets["subnet_ids"].([]interface{}), privateSubnet.ID())
		}

		//Internet Gateway
		gw, err := ec2.NewInternetGateway(ctx, "myInternetGateway", &ec2.InternetGatewayArgs{})
		if err != nil {
			return err
		}

		// Attach gw to the VPC
		_, err = ec2.NewInternetGatewayAttachment(ctx, "exampleInternetGatewayAttachment", &ec2.InternetGatewayAttachmentArgs{
			InternetGatewayId: gw.ID(),
			VpcId:             myVpc.ID(),
		})
		if err != nil {
			return err
		}

		//Public RouteTable
		publicRt, err := ec2.NewRouteTable(ctx, "publicRouteTable", &ec2.RouteTableArgs{
			VpcId: myVpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: gw.ID(),
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("PublicRouteTable"),
			},
		})
		if err != nil {
			return err
		}

		//Private RouteTable
		privateRt, err := ec2.NewRouteTable(ctx, "privateRouteTable", &ec2.RouteTableArgs{
			VpcId: myVpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("PrivateRouteTable"),
			},
		})
		if err != nil {
			return err
		}

		// attach the three public subnets to the public route table
		publicSubnetIDs, ok := publicSubnets["subnet_ids"].([]interface{})
		if !ok {
			return fmt.Errorf("subnet IDs not found in subnets map")
		}

		for idx, publicSubnetID := range publicSubnetIDs {
			associationName := fmt.Sprintf("publicRouteAssociation-%d", idx)

			_, err := ec2.NewRouteTableAssociation(ctx, associationName, &ec2.RouteTableAssociationArgs{
				SubnetId:     publicSubnetID.(pulumi.IDOutput),
				RouteTableId: publicRt.ID(),
			})

			if err != nil {
				return err
			}
		}

		// attach the three private subnets to the private route table
		privateSubnetIDs, ok := privateSubnets["subnet_ids"].([]interface{})
		if !ok {
			return fmt.Errorf("subnet IDs not found in subnets map")
		}

		for idx, privateSubnetID := range privateSubnetIDs {
			associationName := fmt.Sprintf("privateRouteAssociation-%d", idx)

			_, err := ec2.NewRouteTableAssociation(ctx, associationName, &ec2.RouteTableAssociationArgs{
				SubnetId:     privateSubnetID.(pulumi.IDOutput),
				RouteTableId: privateRt.ID(),
			})

			if err != nil {
				return err
			}
		}

		//Create load balancer Security group for RDS instances
		lbSecurityGroup, err := ec2.NewSecurityGroup(ctx, "loadBalancerSecurityGroup", &ec2.SecurityGroupArgs{
			Description: pulumi.String("Load balancer Security Group"),
			VpcId:       myVpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("MariaDB"),
					FromPort:    pulumi.Int(80),
					ToPort:      pulumi.Int(80),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("MariaDB"),
					FromPort:    pulumi.Int(8080),
					ToPort:      pulumi.Int(8080),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("MariaDB"),
					FromPort:    pulumi.Int(443),
					ToPort:      pulumi.Int(443),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("MariaDB database security group"),
			},
		})
		if err != nil {
			return err
		}

		//Create EC2 Security group
		applicationSecurityGroup, err := ec2.NewSecurityGroup(ctx, "applicationSecurityGroup", &ec2.SecurityGroupArgs{
			Description: pulumi.String("Allow TLS inbound traffic"),
			VpcId:       myVpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("TLS from VPC"),
					FromPort:    pulumi.Int(22),
					ToPort:      pulumi.Int(22),
					Protocol:    pulumi.String("tcp"),
					//SecurityGroups: pulumi.StringArray{
					//	lbSecurityGroup.ID(),
					//},
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupIngressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					SecurityGroups: pulumi.StringArray{
						lbSecurityGroup.ID(),
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("allow_tls"),
			},
		})
		if err != nil {
			return err
		}

		//Create EC2 instance
		amiFind, err := ec2.LookupAmi(ctx, &ec2.LookupAmiArgs{
			MostRecent: pulumi.BoolRef(true),
			Filters: []ec2.GetAmiFilter{
				{
					Name: "name",
					Values: []string{
						"csye6225*",
					},
				},
				{
					Name: "virtualization-type",
					Values: []string{
						"hvm",
					},
				},
			},

			Owners: []string{
				"652903061602",
			},
		}, nil)
		if err != nil {
			return err
		}

		//Create DB Security group for RDS instances
		dbSecurityGroup, err := ec2.NewSecurityGroup(ctx, "databaseSecurityGroup", &ec2.SecurityGroupArgs{
			Description: pulumi.String("Database Security Group"),
			VpcId:       myVpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("MariaDB"),
					FromPort:    pulumi.Int(3306),
					ToPort:      pulumi.Int(3306),
					Protocol:    pulumi.String("tcp"),
					SecurityGroups: pulumi.StringArray{
						applicationSecurityGroup.ID(),
					},
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					FromPort: pulumi.Int(0),
					ToPort:   pulumi.Int(0),
					Protocol: pulumi.String("-1"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("MariaDB database security group"),
			},
		})
		if err != nil {
			return err
		}

		//RDS Parameter Group
		rdsParameterGroup, err := rds.NewParameterGroup(ctx, "maria-db-parameter-group", &rds.ParameterGroupArgs{
			Family: pulumi.String("mariadb10.3"),
			Parameters: rds.ParameterGroupParameterArray{
				&rds.ParameterGroupParameterArgs{
					Name:  pulumi.String("character_set_server"),
					Value: pulumi.String("utf8"),
				},
				&rds.ParameterGroupParameterArgs{
					Name:  pulumi.String("character_set_client"),
					Value: pulumi.String("utf8"),
				},
			},
		})
		if err != nil {
			return err
		}

		//create the subnet group for the instance
		rdsSubnetGroup, err := rds.NewSubnetGroup(ctx, "rds subnet group private subnet", &rds.SubnetGroupArgs{
			SubnetIds: pulumi.StringArray{
				privateSubnets["subnet_ids"].([]interface{})[0].(pulumi.IDOutput),
				privateSubnets["subnet_ids"].([]interface{})[1].(pulumi.IDOutput),
				privateSubnets["subnet_ids"].([]interface{})[2].(pulumi.IDOutput),
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("My DB subnet group"),
			},
		})
		if err != nil {
			return err
		}

		// sns topic
		snsTopic, err := sns.NewTopic(ctx, "newTopic", &sns.TopicArgs{
			DisplayName: pulumi.String("csye6225-sns-topic"),
			//	DeliveryPolicy: pulumi.String(`{
			//	  "http": {
			//		"defaultHealthyRetryPolicy": {
			//		  "minDelayTarget": 20,
			//		  "maxDelayTarget": 20,
			//		  "numRetries": 3,
			//		  "numMaxDelayRetries": 0,
			//		  "numNoDelayRetries": 0,
			//		  "numMinDelayRetries": 0,
			//		  "backoffFunction": "linear"
			//		},
			//		"disableSubscriptionOverrides": false,
			//		"defaultThrottlePolicy": {
			//		  "maxReceivesPerSecond": 1
			//		}
			//	  }
			//	}
			//`),
		})

		// RDS instance
		rdsInstance, err := rds.NewInstance(ctx, "rds instance", &rds.InstanceArgs{
			Engine:              pulumi.String("MariaDB"),
			EngineVersion:       pulumi.String("10.3"),
			InstanceClass:       pulumi.String("db.t2.micro"),
			MultiAz:             pulumi.Bool(false),
			Identifier:          pulumi.String("csye6225"),
			Username:            pulumi.String("csye6225"),
			Password:            pulumi.String("LYXliuyixuan0310"),
			DbSubnetGroupName:   rdsSubnetGroup.Name,
			PubliclyAccessible:  pulumi.Bool(false),
			DbName:              pulumi.String("csye6225"),
			AllocatedStorage:    pulumi.Int(10),
			ParameterGroupName:  rdsParameterGroup.Name,
			SkipFinalSnapshot:   pulumi.Bool(true),
			VpcSecurityGroupIds: pulumi.StringArray{dbSecurityGroup.ID()},
		})
		if err != nil {
			return err
		}

		//add iam role - assume the base one
		assumeRole, err := iam.GetPolicyDocument(ctx, &iam.GetPolicyDocumentArgs{
			Statements: []iam.GetPolicyDocumentStatement{
				{
					Effect: pulumi.StringRef("Allow"),
					Principals: []iam.GetPolicyDocumentStatementPrincipal{
						{
							Type: "Service",
							Identifiers: []string{
								"ec2.amazonaws.com",
							},
						},
					},
					Actions: []string{
						"sts:AssumeRole",
					},
				},
			},
		}, nil)

		//create the role for server
		cwServerRole, err := iam.NewRole(ctx, "CloudWatchAgentServerRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(assumeRole.Json),
		})

		//attach the two policies
		_, err = iam.NewRolePolicyAttachment(ctx, "server-role-policy-attach-1", &iam.RolePolicyAttachmentArgs{
			Role:      cwServerRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"),
		})
		if err != nil {
			return err
		}
		_, err = iam.NewRolePolicyAttachment(ctx, "server-role-policy-attach-2", &iam.RolePolicyAttachmentArgs{
			Role:      cwServerRole.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"),
		})
		if err != nil {
			return err
		}

		//create iam user
		cwUserServer, err := iam.NewUser(ctx, "cloudwatch-user-server", nil)
		if err != nil {
			return err
		}

		_, err = iam.NewUserPolicyAttachment(ctx, "user-attach-server-1", &iam.UserPolicyAttachmentArgs{
			User:      cwUserServer.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewUserPolicyAttachment(ctx, "user-attach-server-2", &iam.UserPolicyAttachmentArgs{
			User:      cwUserServer.Name,
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"),
		})
		if err != nil {
			return err
		}

		// Define an IAM instance profile
		instanceProfile, err := iam.NewInstanceProfile(ctx, "ec2InstanceProfile", &iam.InstanceProfileArgs{
			Role: cwServerRole.Name,
		})
		if err != nil {
			return err
		}

		script := `#!/bin/bash
			echo 'DB_ENDPOINT=%s' >> /home/admin/env/properties
			echo 'DB_PASSWORD=%s'  >> /home/admin/env/properties
			echo 'DB_USERNAME=%s' >> /home/admin/env/properties
			echo 'SNS_TOPIC_ARN=%s' >> /home/admin/env/properties
			sudo apt-get update
			sudo apt-get install -y amazon-cloudwatch-agent
			sudo mv /home/admin/cloudwatch-config.json /etc/cloudwatch-config/cloudwatch-config.json
			sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl -a fetch-config -m onPremise -c /etc/cloudwatch-config/cloudwatch-config.json -s
			sudo systemctl enable amazon-cloudwatch-agent
			sudo systemctl start amazon-cloudwatch-agent amazon-cloudwatch-agent
		`
		launchTemplate, err := ec2.NewLaunchTemplate(ctx, "webAppLaunchTemplate", &ec2.LaunchTemplateArgs{
			Name:         pulumi.String("webAppLaunchTemplate"),
			ImageId:      pulumi.String(amiFind.Id),
			InstanceType: pulumi.String("t2.micro"),
			KeyName:      pulumi.String("Macbook"),
			NetworkInterfaces: ec2.LaunchTemplateNetworkInterfaceArray{
				&ec2.LaunchTemplateNetworkInterfaceArgs{
					AssociatePublicIpAddress: pulumi.String("true"),
					SecurityGroups:           pulumi.StringArray{applicationSecurityGroup.ID()},
				},
			},
			UserData: pulumi.Sprintf(script, rdsInstance.Endpoint, "LYXliuyixuan0310", rdsInstance.Username, snsTopic.Arn).ApplyT(scriptBase64).(pulumi.StringOutput),
			//IamInstanceProfile: instanceProfile.Name,
			IamInstanceProfile: &ec2.LaunchTemplateIamInstanceProfileArgs{
				Name: instanceProfile.Name,
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Template Instance"),
			},
			//VpcSecurityGroupIds: pulumi.StringArray{applicationSecurityGroup.ID()},
		})
		if err != nil {
			return err
		}

		//Application Load balancer
		appLoadBalancer, err := lb.NewLoadBalancer(ctx, "test", &lb.LoadBalancerArgs{
			Name:             pulumi.String("appLoadBalancer"),
			Internal:         pulumi.Bool(false),
			LoadBalancerType: pulumi.String("application"),
			SecurityGroups: pulumi.StringArray{
				lbSecurityGroup.ID(),
			},
			Subnets: pulumi.StringArray{
				publicSubnets["subnet_ids"].([]interface{})[0].(pulumi.IDOutput),
				publicSubnets["subnet_ids"].([]interface{})[1].(pulumi.IDOutput),
				publicSubnets["subnet_ids"].([]interface{})[2].(pulumi.IDOutput),
			},
			EnableDeletionProtection: pulumi.Bool(false),
			//AccessLogs: &lb.LoadBalancerAccessLogsArgs{
			//	Bucket:  pulumi.Any(aws_s3_bucket.Lb_logs.Id),
			//	Prefix:  pulumi.String("test-lb"),
			//	Enabled: pulumi.Bool(true),
			//},
			Tags: pulumi.StringMap{
				"Environment": pulumi.String("production"),
			},
		})
		if err != nil {
			return err
		}

		//create a target group
		targetGroup, err := lb.NewTargetGroup(ctx, "port8080", &lb.TargetGroupArgs{
			Port:        pulumi.Int(8080),
			Protocol:    pulumi.String("HTTP"),
			HealthCheck: lb.TargetGroupHealthCheckArgs{Path: pulumi.String("/healthz")},
			VpcId:       myVpc.ID(),
		})
		if err != nil {
			return err
		}

		// auto-scaling group
		autoScalingGroup, err := autoscaling.NewGroup(ctx, "autoScalingGroup", &autoscaling.GroupArgs{
			VpcZoneIdentifiers: pulumi.StringArray{
				publicSubnets["subnet_ids"].([]interface{})[0].(pulumi.IDOutput),
				publicSubnets["subnet_ids"].([]interface{})[1].(pulumi.IDOutput),
				publicSubnets["subnet_ids"].([]interface{})[2].(pulumi.IDOutput),
			},
			DesiredCapacity: pulumi.Int(1),
			DefaultCooldown: pulumi.Int(60),
			MaxSize:         pulumi.Int(3),
			MinSize:         pulumi.Int(1),
			LaunchTemplate: &autoscaling.GroupLaunchTemplateArgs{
				Id:      launchTemplate.ID(),
				Version: pulumi.String("$Latest"),
			},
			Tags: autoscaling.GroupTagArray{
				&autoscaling.GroupTagArgs{
					Key:               pulumi.String("AutoScalingGroup"),
					Value:             pulumi.String("TagProperty"),
					PropagateAtLaunch: pulumi.Bool(true),
				},
			},
			Name:            pulumi.String("autoScalingGroup"),
			TargetGroupArns: pulumi.StringArray{targetGroup.Arn},
			//LoadBalancers:   pulumi.StringArray{appLoadBalancer.Name},
		})
		if err != nil {
			return err
		}

		//AutoScaling up policies
		scaleUpPolicy, err := autoscaling.NewPolicy(ctx, "scaleUpPolicy", &autoscaling.PolicyArgs{
			ScalingAdjustment:    pulumi.Int(1),
			AdjustmentType:       pulumi.String("ChangeInCapacity"),
			Cooldown:             pulumi.Int(60),
			AutoscalingGroupName: autoScalingGroup.Name,
		})
		if err != nil {
			return err
		}

		//create CloudWatch Alarms for CPU utilization
		_, err = cloudwatch.NewMetricAlarm(ctx, "scaleUpAlarm", &cloudwatch.MetricAlarmArgs{
			ComparisonOperator: pulumi.String("GreaterThanThreshold"),
			EvaluationPeriods:  pulumi.Int(2),
			MetricName:         pulumi.String("CPUUtilization"),
			Namespace:          pulumi.String("AWS/EC2"),
			Period:             pulumi.Int(120),
			Statistic:          pulumi.String("Average"),
			Threshold:          pulumi.Float64(5),
			Dimensions: pulumi.StringMap{
				"AutoScalingGroupName": autoScalingGroup.Name,
			},
			AlarmDescription: pulumi.String("This metric monitors EC2 CPU utilization for scaling up"),
			AlarmActions: pulumi.Array{
				scaleUpPolicy.Arn,
			},
		})
		if err != nil {
			return err
		}

		scaleDownPolicy, err := autoscaling.NewPolicy(ctx, "scaleDownPolicy", &autoscaling.PolicyArgs{
			ScalingAdjustment:    pulumi.Int(-1),
			AdjustmentType:       pulumi.String("ChangeInCapacity"),
			Cooldown:             pulumi.Int(60),
			AutoscalingGroupName: autoScalingGroup.Name,
		})
		if err != nil {
			return err
		}
		//create CloudWatch Alarms for CPU utilization
		_, err = cloudwatch.NewMetricAlarm(ctx, "scaleDownAlarm", &cloudwatch.MetricAlarmArgs{
			ComparisonOperator: pulumi.String("LessThanThreshold"),
			EvaluationPeriods:  pulumi.Int(2),
			MetricName:         pulumi.String("CPUUtilization"),
			Namespace:          pulumi.String("AWS/EC2"),
			Period:             pulumi.Int(120),
			Statistic:          pulumi.String("Average"),
			Threshold:          pulumi.Float64(3),
			Dimensions: pulumi.StringMap{
				"AutoScalingGroupName": autoScalingGroup.Name,
			},
			AlarmDescription: pulumi.String("This metric monitors EC2 CPU utilization for scaling down"),
			AlarmActions: pulumi.Array{
				scaleDownPolicy.Arn,
			},
		})
		if err != nil {
			return err
		}

		//create a listener
		listener, err := lb.NewListener(ctx, "frontEndListener", &lb.ListenerArgs{
			LoadBalancerArn: appLoadBalancer.Arn,
			//Port:            pulumi.Int(80),
			//Protocol:        pulumi.String("HTTP"),
			Port:           pulumi.Int(443),
			Protocol:       pulumi.String("HTTPS"),
			CertificateArn: pulumi.String("arn:aws:acm:us-west-2:652903061602:certificate/707a9209-49b6-4e86-a8cf-23da50fcf72b"),
			//SslPolicy:      pulumi.String("ELBSecurityPolicy-2016-08"),
			//CertificateArn: pulumi.String("arn:aws:iam::187416307283:server-certificate/test_cert_rab3wuqwgja25ct3n4jdj2tzu4"),
			DefaultActions: lb.ListenerDefaultActionArray{
				&lb.ListenerDefaultActionArgs{
					Type:           pulumi.String("forward"),
					TargetGroupArn: targetGroup.Arn,
				},
			},
		})
		if err != nil {
			return err
		}

		//_, err = lb.NewListenerCertificate(ctx, "exampleListenerCertificate", &lb.ListenerCertificateArgs{
		//	ListenerArn:    listener.Arn,
		//	CertificateArn: pulumi.String("arn:aws:acm:us-west-2:652903061602:certificate/707a9209-49b6-4e86-a8cf-23da50fcf72b"),
		//})

		// Create a listener rule to forward traffic to the target group
		_, err = lb.NewListenerRule(ctx, "test-listener-rule", &lb.ListenerRuleArgs{
			ListenerArn: listener.Arn,
			Priority:    pulumi.Int(1),
			Conditions: lb.ListenerRuleConditionArray{
				&lb.ListenerRuleConditionArgs{
					PathPattern: &lb.ListenerRuleConditionPathPatternArgs{
						Values: pulumi.StringArray{
							pulumi.String("/*"),
						},
					},
				},
			},
			Actions: lb.ListenerRuleActionArray{
				&lb.ListenerRuleActionArgs{
					Type:           pulumi.String("forward"),
					TargetGroupArn: targetGroup.Arn,
				},
			},
		})
		if err != nil {
			return err
		}

		//add a route53 record
		_, err = route53.NewRecord(ctx, "dev.cecilialiu.cc", &route53.RecordArgs{
			ZoneId: pulumi.String("Z04798253J5N87ZG4JUET"), //dev:Z04798253J5N87ZG4JUET root:Z06716922NC0RFI5IJNM1
			Name:   pulumi.String("dev.cecilialiu.cc"),
			Type:   pulumi.String("A"),
			//Ttl:    pulumi.Int(172800),
			//Records: pulumi.StringArray{
			//	myEc2.PublicIp, //aws_eip.Lb.Public_ip
			//},
			Aliases: route53.RecordAliasArray{
				&route53.RecordAliasArgs{
					Name:                 appLoadBalancer.DnsName,
					ZoneId:               appLoadBalancer.ZoneId,
					EvaluateTargetHealth: pulumi.Bool(true),
				},
			},
		})
		if err != nil {
			return err
		}

		//bucket gcs-bucket
		gcsBucket, err := storage.NewBucket(ctx, "gcs-bucket-csye6225", &storage.BucketArgs{
			Cors: storage.BucketCorArray{
				&storage.BucketCorArgs{
					MaxAgeSeconds: pulumi.Int(3600),
					//Methods: pulumi.StringArray{
					//	pulumi.String("GET"),
					//	pulumi.String("HEAD"),
					//	pulumi.String("PUT"),
					//	pulumi.String("POST"),
					//	pulumi.String("DELETE"),
					//},
				},
			},
			ForceDestroy:             pulumi.Bool(true),
			Location:                 pulumi.String("US"),
			PublicAccessPrevention:   pulumi.String("enforced"),
			UniformBucketLevelAccess: pulumi.Bool(false),
		})

		//google service account
		serviceAccount, err := serviceaccount.NewAccount(ctx, "serviceAccount", &serviceaccount.AccountArgs{
			AccountId:   pulumi.String("gcs-service-account"),
			DisplayName: pulumi.String("Service Account"),
			Project:     pulumi.String("dev-csye6225-406007"),
		})
		if err != nil {
			return err
		}

		//_, err = serviceaccount.NewIAMBinding(ctx, "admin-account-iam-1", &serviceaccount.IAMBindingArgs{
		//	ServiceAccountId: serviceAccount.Name,
		//	Role:             pulumi.String("roles/storage.admin"),
		//})
		//if err != nil {
		//	return err
		//}

		_, err = storage.NewBucketIAMBinding(ctx, "binding-1", &storage.BucketIAMBindingArgs{
			Bucket: gcsBucket.Name,
			Role:   pulumi.String("roles/storage.admin"),
			Members: pulumi.StringArray{
				pulumi.String("user:cecillyaliu@gmail.com"),
			},
		})
		if err != nil {
			return err
		}

		_, err = storage.NewBucketIAMBinding(ctx, "binding-2", &storage.BucketIAMBindingArgs{
			Bucket: gcsBucket.Name,
			Role:   pulumi.String("roles/storage.objectUser"),
			Members: pulumi.StringArray{
				pulumi.String("user:cecillyaliu@gmail.com"),
			},
		})
		if err != nil {
			return err
		}

		//_, err = serviceaccount.NewIAMBinding(ctx, "admin-account-iam-2", &serviceaccount.IAMBindingArgs{
		//	ServiceAccountId: serviceAccount.Name,
		//	Role:             pulumi.String("roles/storage.objectUser"),
		//	Members: pulumi.StringArray{
		//		pulumi.String("user:cecillyaliu@gmail.com"),
		//	},
		//})
		if err != nil {
			return err
		}

		//access keys
		accessKey, err := serviceaccount.NewKey(ctx, "accessKey", &serviceaccount.KeyArgs{
			ServiceAccountId: serviceAccount.Name,
			PublicKeyType:    pulumi.String("TYPE_X509_PEM_FILE"),
		})
		if err != nil {
			return err
		}

		//DynamoDB table
		dynamoTable, err := dynamodb.NewTable(ctx, "myDynamoTable", &dynamodb.TableArgs{
			Attributes: dynamodb.TableAttributeArray{
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("id"),
					Type: pulumi.String("S"),
				},
				&dynamodb.TableAttributeArgs{
					Name: pulumi.String("date_time"),
					Type: pulumi.String("S"),
				},
			},
			RangeKey: pulumi.String("date_time"),
			//GlobalSecondaryIndexes: dynamodb.TableGlobalSecondaryIndexArray{
			//	&dynamodb.TableGlobalSecondaryIndexArgs{
			//		HashKey: pulumi.String("GameTitle"),
			//		Name:    pulumi.String("GameTitleIndex"),
			//		NonKeyAttributes: pulumi.StringArray{
			//			pulumi.String("UserId"),
			//		},
			//		ProjectionType: pulumi.String("INCLUDE"),
			//		RangeKey:       pulumi.String("TopScore"),
			//		ReadCapacity:   pulumi.Int(10),
			//		WriteCapacity:  pulumi.Int(10),
			//	},
			//},
			HashKey:       pulumi.String("id"),
			BillingMode:   pulumi.String("PROVISIONED"),
			ReadCapacity:  pulumi.Int(20),
			WriteCapacity: pulumi.Int(20),
			Tags: pulumi.StringMap{
				"Environment": pulumi.String("production"),
				"Name":        pulumi.String("dynamodb-table-1"),
			},
		})
		if err != nil {
			return err
		}

		// iam roles and policies for lambda function
		role, err := iam.NewRole(ctx, "LambdaRole", &iam.RoleArgs{
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [
					{
						"Effect": "Allow",
						"Principal": {
							"Service": "lambda.amazonaws.com"
						},
						"Action": "sts:AssumeRole"
					}
				]
			}`),
			Tags: pulumi.StringMap{
				"Name:": pulumi.String("Lambda iam role"),
			},
		})
		if err != nil {
			return err
		}

		_, err = iam.NewPolicyAttachment(ctx, "attach-lambda-iam-role-1", &iam.PolicyAttachmentArgs{
			Roles: pulumi.Array{
				role.Name,
			},
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonDynamoDBFullAccess"),
		})

		_, err = iam.NewPolicyAttachment(ctx, "attach-lambda-iam-role-2", &iam.PolicyAttachmentArgs{
			Roles: pulumi.Array{
				role.Name,
			},
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonSESFullAccess"),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewPolicyAttachment(ctx, "attach-lambda-iam-role-3", &iam.PolicyAttachmentArgs{
			Roles: pulumi.Array{
				role.Name,
			},
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonSNSFullAccess"),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewPolicyAttachment(ctx, "attach-lambda-iam-role-4", &iam.PolicyAttachmentArgs{
			Roles: pulumi.Array{
				role.Name,
			},
			PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
		})
		if err != nil {
			return err
		}

		//Store email server configuration
		_, err = secretsmanager.NewSecret(ctx, "mySecret", &secretsmanager.SecretArgs{
			Description: pulumi.String("Email Server Configuration"),
		})
		if err != nil {
			return err
		}

		layerLambda, err := lambda.NewLayerVersion(ctx, "gcs-layer", &lambda.LayerVersionArgs{
			CompatibleRuntimes: pulumi.StringArray{
				pulumi.String("python3.9"),
			},
			Code:      pulumi.NewFileArchive("aws-layer-gcs.zip"),
			LayerName: pulumi.String("gcs-layer"),
		})
		if err != nil {
			return err
		}

		//Configure Lambda Function with Google Access Keys and bucket name
		lambdaFunction, err := lambda.NewFunction(ctx, "Lambda_A9", &lambda.FunctionArgs{
			Code:    pulumi.NewFileArchive("./lambda_codes.zip"),
			Role:    role.Arn,
			Handler: pulumi.String("lambda_function.lambda_handler"),
			Runtime: pulumi.String("python3.9"),
			Layers: pulumi.StringArray{
				layerLambda.Arn,
			},
			Timeout: pulumi.Int(60),
			Environment: &lambda.FunctionEnvironmentArgs{
				Variables: pulumi.StringMap{
					"google_token":        accessKey.PrivateKey,
					"github_token":        pulumi.String("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"),
					"BUCKET_NAME":         gcsBucket.Name,
					"DYNAMODB_TABLE_NAME": dynamoTable.Name,
					"RECIPIENT":           pulumi.String("cecillyaliu@gmail.com"),
					//"EMAIL_SERVER":             pulumi.String("smtp.example.com"),
					//"EMAIL_USER":                     pulumi.String("user"),
					//"EMAIL_CONFIGURATION_SECRET_ARN": emailSecret.Arn,
				},
			},
		})
		if err != nil {
			return err
		}
		//lambda function layer

		//sns trigger lambda
		_, err = sns.NewTopicSubscription(ctx, "userUpdatesSqsTarget", &sns.TopicSubscriptionArgs{
			Endpoint: lambdaFunction.Arn,
			Protocol: pulumi.String("lambda"),
			Topic:    snsTopic.Arn,
		})
		if err != nil {
			return err
		}

		return nil
	})

}

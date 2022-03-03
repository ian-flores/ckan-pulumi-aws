package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

type VpcConfig struct {
	CidrBlock              string
	PublicSubnetCidrBlock  string
	PrivateSubnetCidrBlock string
}

func errorHandler(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		VpcConfig := &VpcConfig{}
		conf := config.New(ctx, "")
		conf.RequireObject("Vpc", &VpcConfig)

		// Create a VPC
		vpc, err := ec2.NewVpc(ctx, "ckan-pulumi-vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String(VpcConfig.CidrBlock),
		})

		errorHandler(err)

		// Create the Internet Gateway
		igw, err := ec2.NewInternetGateway(ctx, "ckan-pulumi-igw", &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
		})

		errorHandler(err)

		// Create the main route table
		rt, err := ec2.NewRouteTable(ctx, "ckan-pulumi-rt", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					// Connect to the internet through the Internet Gateway
					// If the IP is not within the CIDR block range of the VPC
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: igw.ID(),
				},
			},
		})

		errorHandler(err)

		// Create the public subnet
		publicSubnet, err := ec2.NewSubnet(ctx, "ckan-pulumi-public-subnet", &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String(VpcConfig.PublicSubnetCidrBlock),
			MapPublicIpOnLaunch: pulumi.Bool(true),
		})

		errorHandler(err)

		// Create the public subnet <==> route table association
		_, err = ec2.NewRouteTableAssociation(ctx, "ckan-pulumi-public-subnet-rt-assoc", &ec2.RouteTableAssociationArgs{
			RouteTableId: rt.ID(),
			SubnetId:     publicSubnet.ID(),
		})

		errorHandler(err)

		// Create the public subnet NACL

		publicSubnetNacl, err := ec2.NewNetworkAcl(ctx, "ckan-pulumi-public-subnet-nacl", &ec2.NetworkAclArgs{
			VpcId: vpc.ID(),
			Egress: ec2.NetworkAclEgressArray{
				&ec2.NetworkAclEgressArgs{
					Protocol:  pulumi.String("tcp"),
					Action:    pulumi.String("allow"),
					CidrBlock: pulumi.String("0.0.0.0/0"),
					RuleNo:    pulumi.Int(100),
					FromPort:  pulumi.Int(0),
					ToPort:    pulumi.Int(0),
				},
				&ec2.NetworkAclEgressArgs{
					Protocol:  pulumi.String("-1"),
					Action:    pulumi.String("deny"),
					CidrBlock: pulumi.String("0.0.0.0/0"),
					RuleNo:    pulumi.Int(50),
					FromPort:  pulumi.Int(0),
					ToPort:    pulumi.Int(0),
				},
			},
			Ingress: ec2.NetworkAclIngressArray{
				&ec2.NetworkAclIngressArgs{
					Protocol:  pulumi.String("tcp"),
					Action:    pulumi.String("allow"),
					CidrBlock: pulumi.String("0.0.0.0/0"),
					RuleNo:    pulumi.Int(100),
					FromPort:  pulumi.Int(80),
					ToPort:    pulumi.Int(80),
				},
				&ec2.NetworkAclIngressArgs{
					Protocol:  pulumi.String("-1"),
					Action:    pulumi.String("deny"),
					CidrBlock: pulumi.String("0.0.0.0/0"),
					RuleNo:    pulumi.Int(50),
					FromPort:  pulumi.Int(0),
					ToPort:    pulumi.Int(0),
				},
			},
			SubnetIds: pulumi.StringArray{publicSubnet.ID()},
		})

		errorHandler(err)

		// Create the private subnet
		privateSubnet, err := ec2.NewSubnet(ctx, "ckan-pulumi-private-subnet", &ec2.SubnetArgs{
			VpcId:               vpc.ID(),
			CidrBlock:           pulumi.String(VpcConfig.PrivateSubnetCidrBlock),
			MapPublicIpOnLaunch: pulumi.Bool(false),
		})

		errorHandler(err)

		// Create the private subnet <==> route table association
		_, err = ec2.NewRouteTableAssociation(ctx, "ckan-pulumi-private-subnet-rt-assoc", &ec2.RouteTableAssociationArgs{
			RouteTableId: rt.ID(),
			SubnetId:     privateSubnet.ID(),
		})

		errorHandler(err)

		// Create the private subnet NACL
		privateSubnetNacl, err := ec2.NewNetworkAcl(ctx, "ckan-pulumi-private-subnet-nacl", &ec2.NetworkAclArgs{
			VpcId: vpc.ID(),
			Egress: ec2.NetworkAclEgressArray{
				&ec2.NetworkAclEgressArgs{
					Protocol:  pulumi.String("tcp"),
					Action:    pulumi.String("allow"),
					CidrBlock: pulumi.String(VpcConfig.PublicSubnetCidrBlock),
					RuleNo:    pulumi.Int(100),
					FromPort:  pulumi.Int(0),
					ToPort:    pulumi.Int(0),
				},
				&ec2.NetworkAclEgressArgs{
					Protocol:  pulumi.String("-1"),
					Action:    pulumi.String("deny"),
					CidrBlock: pulumi.String("0.0.0.0/0"),
					RuleNo:    pulumi.Int(50),
					FromPort:  pulumi.Int(0),
					ToPort:    pulumi.Int(0),
				},
			},
			Ingress: ec2.NetworkAclIngressArray{
				&ec2.NetworkAclIngressArgs{
					Protocol:  pulumi.String("tcp"),
					Action:    pulumi.String("allow"),
					CidrBlock: pulumi.String(VpcConfig.PublicSubnetCidrBlock),
					RuleNo:    pulumi.Int(100),
					FromPort:  pulumi.Int(5432),
					ToPort:    pulumi.Int(5432),
				},
				&ec2.NetworkAclIngressArgs{
					Protocol:  pulumi.String("-1"),
					Action:    pulumi.String("deny"),
					CidrBlock: pulumi.String("0.0.0.0/0"),
					RuleNo:    pulumi.Int(50),
					FromPort:  pulumi.Int(0),
					ToPort:    pulumi.Int(0),
				},
			},
			SubnetIds: pulumi.StringArray{privateSubnet.ID()},
		})

		errorHandler(err)

		// Create the security group for the RDS instance
		rdsSG, err := ec2.NewSecurityGroup(ctx, "ckan-pulumi-rds-sg", &ec2.SecurityGroupArgs{
			VpcId: vpc.ID(),
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					CidrBlocks: pulumi.StringArray{
						pulumi.String(VpcConfig.PublicSubnetCidrBlock),
					},
					FromPort:    pulumi.Int(0),
					ToPort:      pulumi.Int(0),
					Protocol:    pulumi.String("-1"),
					Description: pulumi.String("Allow all outbound traffic to the public subnet"),
				},
			},
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					CidrBlocks: pulumi.StringArray{
						pulumi.String(VpcConfig.PublicSubnetCidrBlock),
					},
					FromPort:    pulumi.Int(5432),
					ToPort:      pulumi.Int(5432),
					Protocol:    pulumi.String("tcp"),
					Description: pulumi.String("Allow access to the RDS instance from the public subnet"),
				},
			},
		})

		errorHandler(err)

		// Create the security group for the web server
		webSG, err := ec2.NewSecurityGroup(ctx, "ckan-pulumi-web-sg", &ec2.SecurityGroupArgs{
			VpcId: vpc.ID(),
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
					FromPort:    pulumi.Int(0),
					ToPort:      pulumi.Int(0),
					Protocol:    pulumi.String("-1"),
					Description: pulumi.String("Allow all outbound traffic"),
				},
			},
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
					FromPort:    pulumi.Int(80),
					ToPort:      pulumi.Int(80),
					Protocol:    pulumi.String("tcp"),
					Description: pulumi.String("Allow access to the web server from the internet"),
				},
			},
		})

		errorHandler(err)

		ctx.Export("vpc", vpc.ID())
		ctx.Export("igw", igw.ID())
		ctx.Export("rt", rt.ID())
		ctx.Export("publicSubnet", publicSubnet.ID())
		ctx.Export("publicSubnetNacl", publicSubnetNacl.ID())
		ctx.Export("privateSubnet", privateSubnet.ID())
		ctx.Export("privateSubnetNacl", privateSubnetNacl.ID())
		ctx.Export("rdsSG", rdsSG.ID())
		ctx.Export("webSG", webSG.ID())

		return nil
	})
}

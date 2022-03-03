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

		ctx.Export("vpc", vpc.ID())
		ctx.Export("igw", igw.ID())
		ctx.Export("rt", rt.ID())
		ctx.Export("publicSubnet", publicSubnet.ID())
		ctx.Export("privateSubnet", privateSubnet.ID())

		return nil
	})
}

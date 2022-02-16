/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	armnetwork "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork"
	armresources "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/go-autorest/autorest/to"
)

var (
	subscriptionID      string
	location            = "westus2"
	prefix              = "mirsik" // Replace with your alias.
	resourceGroupName   = prefix + "-rg"
	virtualNetworkName  = prefix + "-vn"
	subnetName          = prefix + "-subnet"
	vmScaleSetName      = prefix + "-vmss"
	bastionHostName     = prefix + "-bhost"
	publicIPAddressName = prefix + "-ip"
	username            = "azureuser"
)

func main() {
	subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	resourceGroup, err := createResourceGroup(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("resources group:", *resourceGroup.ID)

	virtualNetwork, err := createVirtualNetwork(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("virtual network:", *virtualNetwork.ID)

	subnet, err := createSubnet(ctx, cred, subnetName, "10.1.0.0/24")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("subnet:", *subnet.ID)

	password := "A5" + RandStringBytes(10)
	vmss, err := createVMSS(ctx, cred, *subnet.ID, username, password)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("virtual machine scale sets:", *vmss.ID)

	publicIP, err := createPublicIP(ctx, cred)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("public IP:", *publicIP.ID)

	bastionsubnet, err := createSubnet(ctx, cred, "AzureBastionSubnet", "10.1.1.0/24")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("subnet:", *subnet.ID)

	bastionHost, err := createBastion(ctx, cred, *bastionsubnet.ID, *publicIP.ID)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Bastion host:", *bastionHost.ID)

	log.Println()
	log.Println("Use this to login using bastion:")
	log.Println("Username: ", username, " password: ", password)
}

func createResourceGroup(ctx context.Context, cred azcore.TokenCredential) (*armresources.ResourceGroup, error) {
	resourceGroupClient := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)

	resourceGroupResp, err := resourceGroupClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		armresources.ResourceGroup{
			Location: to.StringPtr(location),
		},
		nil)
	if err != nil {
		return nil, err
	}
	return &resourceGroupResp.ResourceGroup, nil
}

func createVirtualNetwork(ctx context.Context, cred azcore.TokenCredential) (*armnetwork.VirtualNetwork, error) {
	virtualNetworkClient := armnetwork.NewVirtualNetworksClient(subscriptionID, cred, nil)

	pollerResp, err := virtualNetworkClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		virtualNetworkName,
		armnetwork.VirtualNetwork{
			Location: to.StringPtr(location),
			Properties: &armnetwork.VirtualNetworkPropertiesFormat{
				AddressSpace: &armnetwork.AddressSpace{
					AddressPrefixes: []*string{
						to.StringPtr("10.1.0.0/16"),
					},
				},
			},
		},
		nil)

	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.VirtualNetwork, nil
}

func createSubnet(ctx context.Context, cred azcore.TokenCredential, subnetName string, addressPrefix string) (*armnetwork.Subnet, error) {
	subnetsClient := armnetwork.NewSubnetsClient(subscriptionID, cred, nil)

	pollerResp, err := subnetsClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		virtualNetworkName,
		subnetName,
		armnetwork.Subnet{
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr(addressPrefix),
			},
		},
		nil)

	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.Subnet, nil
}

func createVMSS(ctx context.Context, cred azcore.TokenCredential, subnetID string, username string, password string) (*armcompute.VirtualMachineScaleSet, error) {
	vmssClient := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, cred, nil)

	pollerResp, err := vmssClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		vmScaleSetName,
		armcompute.VirtualMachineScaleSet{
			Location: to.StringPtr(location),
			SKU: &armcompute.SKU{
				Name:     to.StringPtr("Basic_A0"), //armcompute.VirtualMachineSizeTypesBasicA0
				Capacity: to.Int64Ptr(1),
			},
			Properties: &armcompute.VirtualMachineScaleSetProperties{
				Overprovision: to.BoolPtr(false),
				UpgradePolicy: &armcompute.UpgradePolicy{
					Mode: armcompute.UpgradeModeManual.ToPtr(),
					AutomaticOSUpgradePolicy: &armcompute.AutomaticOSUpgradePolicy{
						EnableAutomaticOSUpgrade: to.BoolPtr(false),
						DisableAutomaticRollback: to.BoolPtr(false),
					},
				},
				VirtualMachineProfile: &armcompute.VirtualMachineScaleSetVMProfile{
					OSProfile: &armcompute.VirtualMachineScaleSetOSProfile{
						ComputerNamePrefix: to.StringPtr("vmss"),
						AdminUsername:      to.StringPtr(username),
						AdminPassword:      to.StringPtr(password),
					},
					StorageProfile: &armcompute.VirtualMachineScaleSetStorageProfile{
						ImageReference: &armcompute.ImageReference{
							Offer:     to.StringPtr("UbuntuServer"),
							Publisher: to.StringPtr("Canonical"),
							SKU:       to.StringPtr("18.04-LTS"),
							Version:   to.StringPtr("latest"),
						},
					},
					NetworkProfile: &armcompute.VirtualMachineScaleSetNetworkProfile{
						NetworkInterfaceConfigurations: []*armcompute.VirtualMachineScaleSetNetworkConfiguration{
							{
								Name: to.StringPtr(vmScaleSetName),
								Properties: &armcompute.VirtualMachineScaleSetNetworkConfigurationProperties{
									Primary:            to.BoolPtr(true),
									EnableIPForwarding: to.BoolPtr(true),
									IPConfigurations: []*armcompute.VirtualMachineScaleSetIPConfiguration{
										{
											Name: to.StringPtr(vmScaleSetName),
											Properties: &armcompute.VirtualMachineScaleSetIPConfigurationProperties{
												Subnet: &armcompute.APIEntityReference{
													ID: to.StringPtr(subnetID),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.VirtualMachineScaleSet, nil
}

func createPublicIP(ctx context.Context, cred azcore.TokenCredential) (*armnetwork.PublicIPAddress, error) {
	publicIPClient := armnetwork.NewPublicIPAddressesClient(subscriptionID, cred, nil)

	pollerResp, err := publicIPClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		publicIPAddressName,
		armnetwork.PublicIPAddress{
			Name:     to.StringPtr(publicIPAddressName),
			Location: to.StringPtr(location),
			Properties: &armnetwork.PublicIPAddressPropertiesFormat{
				PublicIPAllocationMethod: armnetwork.IPAllocationMethodStatic.ToPtr(),
			},
			SKU: &armnetwork.PublicIPAddressSKU{
				Name: armnetwork.PublicIPAddressSKUNameStandard.ToPtr(),
			},
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.PublicIPAddress, nil
}

func createBastion(ctx context.Context, cred azcore.TokenCredential, subnetID string, publicIpId string) (*armnetwork.BastionHost, error) {
	bastionHostClient := armnetwork.NewBastionHostsClient(subscriptionID, cred, nil)

	pollerResp, err := bastionHostClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		virtualNetworkName,
		armnetwork.BastionHost{
			Location: &location,
			Name:     &bastionHostName,
			Properties: &armnetwork.BastionHostPropertiesFormat{
				IPConfigurations: []*armnetwork.BastionHostIPConfiguration{
					{
						Name: to.StringPtr("IpConf"),
						Properties: &armnetwork.BastionHostIPConfigurationPropertiesFormat{
							PublicIPAddress: &armnetwork.SubResource{
								ID: &publicIpId,
							},
							Subnet: &armnetwork.SubResource{
								ID: &subnetID,
							},
						},
					},
				},
			},
		},
		nil)

	if err != nil {
		return nil, err
	}

	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.BastionHost, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

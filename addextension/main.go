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
	"encoding/base64"
	"log"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armcompute "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"
	"github.com/Azure/go-autorest/autorest/to"
)

var (
	subscriptionID string
	prefix         string
	version        = 1 // Update this to rerun the extension.
)

func GenerateScript() string {
	commands := `
filepath="/tmp/test-$(date +"%m-%d-%Y-%T").log"
echo "hello" > $filepath
for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do
  sleep 5
  echo "$i: waited 5 secs" >> $filepath
done`
	return base64.StdEncoding.EncodeToString([]byte(commands))
}

func main() {
	subscriptionID = os.Getenv("AZURE_SUBSCRIPTION_ID")
	if len(subscriptionID) == 0 {
		log.Fatal("AZURE_SUBSCRIPTION_ID is not set.")
	}

	prefix = os.Getenv("MS_ALIAS")
	if len(subscriptionID) == 0 {
		log.Fatal("MS_ALIAS is not set.")
	}

	resourceGroupName := prefix + "-rg"
	vmScaleSetName := prefix + "-vmss"
	vmScaleSetExtName := prefix + "-vmssExt"

	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	vmssExt, err := AddExtension(ctx, cred, resourceGroupName, vmScaleSetName, vmScaleSetExtName)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("virtual machine scale set extension updated:", *vmssExt.ID)

	status, err := UpgradeInstance(ctx, cred, resourceGroupName, vmScaleSetName)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("virtual machine scale set vm updated:", *status)

}

func AddExtension(ctx context.Context, cred azcore.TokenCredential, resourceGroupName string, vmScaleSetName string, vmScaleSetExtName string) (*armcompute.VirtualMachineScaleSetExtension, error) {
	vmssExtClient := armcompute.NewVirtualMachineScaleSetExtensionsClient(subscriptionID, cred, nil)

	pollerResp, err := vmssExtClient.BeginCreateOrUpdate(
		ctx,
		resourceGroupName,
		vmScaleSetName,
		vmScaleSetExtName,
		armcompute.VirtualMachineScaleSetExtension{
			Properties: &armcompute.VirtualMachineScaleSetExtensionProperties{
				Publisher:               to.StringPtr("Microsoft.Azure.Extensions"),
				Type:                    to.StringPtr("CustomScript"),
				TypeHandlerVersion:      to.StringPtr("2.1"),
				AutoUpgradeMinorVersion: to.BoolPtr(true),
				ProtectedSettings: map[string]interface{}{
					"script": GenerateScript(),
				},
				ForceUpdateTag: to.StringPtr(string(version)),
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
	return &resp.VirtualMachineScaleSetExtension, nil
}

func UpgradeInstance(ctx context.Context, cred azcore.TokenCredential, resourceGroupName string, vmScaleSetName string) (*string, error) {
	vmssVmsClient := armcompute.NewVirtualMachineScaleSetsClient(subscriptionID, cred, nil)

	pollerResp, err := vmssVmsClient.BeginUpdateInstances(ctx, resourceGroupName, vmScaleSetName, armcompute.VirtualMachineScaleSetVMInstanceRequiredIDs{InstanceIDs: []*string{to.StringPtr("0")}}, nil)
	if err != nil {
		return nil, err
	}
	resp, err := pollerResp.PollUntilDone(ctx, 10*time.Second)
	if err != nil {
		return nil, err
	}
	return &resp.RawResponse.Status, nil
}

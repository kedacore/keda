#!/usr/bin/env pwsh
  
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License.

# This script will bootstrap your .env file, given a resource group that's been deployed to.
# We expect a premium and a standard Service Bus namespace for our tests.
#
# You might need to install the Azure Powershell module first and login:
#
# Install-Module -Name Az -Repository PSGallery -Force
# Connect-AzAccount -UseDeviceAuthentication
# 

# TODO:
$rg = "<your resource group should go here>"

$contents = ""

Get-AzServiceBusNamespace -ResourceGroup $rg
| ForEach-Object {
    $cs = (Get-AzServiceBusKey -ResourceGroup $rg -NamespaceName $_.Name -AuthorizationRuleName RootManageSharedAccessKey).PrimaryConnectionString
    $endpoint = $_.ServiceBusEndpoint.Replace("https://", "").Replace(":443/", "")

    if ($_.SkuTier -eq "Standard") {
        # this is in the .bicep file for this - we create a few keys with different permissions for testing.
        $noManageCs = (Get-AzServiceBusKey -ResourceGroup $rg -NamespaceName $_.Name -AuthorizationRuleName NoManage).PrimaryConnectionString
        $sendOnlyCs = (Get-AzServiceBusKey -ResourceGroup $rg -NamespaceName $_.Name -AuthorizationRuleName SendOnly).PrimaryConnectionString
        $listenOnlyCs = (Get-AzServiceBusKey -ResourceGroup $rg -NamespaceName $_.Name -AuthorizationRuleName ListenOnly).PrimaryConnectionString

        $contents += "`n# standard`n"
        $contents += "SERVICEBUS_CONNECTION_STRING=$cs`n"
        $contents += "SERVICEBUS_ENDPOINT=$endpoint`n"
        $contents += "SERVICEBUS_CONNECTION_STRING_NO_MANAGE=$noManageCs`n"
        $contents += "SERVICEBUS_CONNECTION_STRING_SEND_ONLY=$sendOnlyCs`n"
        $contents += "SERVICEBUS_CONNECTION_STRING_LISTEN_ONLY=$listenOnlyCs`n"
    }
    else {
        # we do a little bit of testing on premium.
        $contents += "`n# premium`n"
        $contents += "SERVICEBUS_CONNECTION_STRING_PREMIUM=$cs`n"
        $contents += "SERVICEBUS_ENDPOINT_PREMIUM=$endpoint`n"
    }    
} 

Out-File ".env" -InputObject $contents

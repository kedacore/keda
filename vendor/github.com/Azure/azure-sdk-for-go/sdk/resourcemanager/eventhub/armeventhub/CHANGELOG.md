# Release History

## 1.3.0 (2024-07-24)
### Features Added

- New value `PublicNetworkAccessFlagSecuredByPerimeter` added to enum type `PublicNetworkAccessFlag`
- New enum type `ApplicationGroupPolicyType` with values `ApplicationGroupPolicyTypeThrottlingPolicy`
- New enum type `CaptureIdentityType` with values `CaptureIdentityTypeSystemAssigned`, `CaptureIdentityTypeUserAssigned`
- New enum type `CleanupPolicyRetentionDescription` with values `CleanupPolicyRetentionDescriptionCompact`, `CleanupPolicyRetentionDescriptionDelete`
- New enum type `MetricID` with values `MetricIDIncomingBytes`, `MetricIDIncomingMessages`, `MetricIDOutgoingBytes`, `MetricIDOutgoingMessages`
- New enum type `NetworkSecurityPerimeterConfigurationProvisioningState` with values `NetworkSecurityPerimeterConfigurationProvisioningStateAccepted`, `NetworkSecurityPerimeterConfigurationProvisioningStateCanceled`, `NetworkSecurityPerimeterConfigurationProvisioningStateCreating`, `NetworkSecurityPerimeterConfigurationProvisioningStateDeleted`, `NetworkSecurityPerimeterConfigurationProvisioningStateDeleting`, `NetworkSecurityPerimeterConfigurationProvisioningStateFailed`, `NetworkSecurityPerimeterConfigurationProvisioningStateInvalidResponse`, `NetworkSecurityPerimeterConfigurationProvisioningStateSucceeded`, `NetworkSecurityPerimeterConfigurationProvisioningStateSucceededWithIssues`, `NetworkSecurityPerimeterConfigurationProvisioningStateUnknown`, `NetworkSecurityPerimeterConfigurationProvisioningStateUpdating`
- New enum type `NspAccessRuleDirection` with values `NspAccessRuleDirectionInbound`, `NspAccessRuleDirectionOutbound`
- New enum type `ProvisioningState` with values `ProvisioningStateActive`, `ProvisioningStateCanceled`, `ProvisioningStateCreating`, `ProvisioningStateDeleting`, `ProvisioningStateFailed`, `ProvisioningStateScaling`, `ProvisioningStateSucceeded`, `ProvisioningStateUnknown`
- New enum type `PublicNetworkAccess` with values `PublicNetworkAccessDisabled`, `PublicNetworkAccessEnabled`, `PublicNetworkAccessSecuredByPerimeter`
- New enum type `ResourceAssociationAccessMode` with values `ResourceAssociationAccessModeAuditMode`, `ResourceAssociationAccessModeEnforcedMode`, `ResourceAssociationAccessModeLearningMode`, `ResourceAssociationAccessModeNoAssociationMode`, `ResourceAssociationAccessModeUnspecifiedMode`
- New enum type `TLSVersion` with values `TLSVersionOne0`, `TLSVersionOne1`, `TLSVersionOne2`
- New function `NewApplicationGroupClient(string, azcore.TokenCredential, *arm.ClientOptions) (*ApplicationGroupClient, error)`
- New function `*ApplicationGroupClient.CreateOrUpdateApplicationGroup(context.Context, string, string, string, ApplicationGroup, *ApplicationGroupClientCreateOrUpdateApplicationGroupOptions) (ApplicationGroupClientCreateOrUpdateApplicationGroupResponse, error)`
- New function `*ApplicationGroupClient.Delete(context.Context, string, string, string, *ApplicationGroupClientDeleteOptions) (ApplicationGroupClientDeleteResponse, error)`
- New function `*ApplicationGroupClient.Get(context.Context, string, string, string, *ApplicationGroupClientGetOptions) (ApplicationGroupClientGetResponse, error)`
- New function `*ApplicationGroupClient.NewListByNamespacePager(string, string, *ApplicationGroupClientListByNamespaceOptions) *runtime.Pager[ApplicationGroupClientListByNamespaceResponse]`
- New function `*ApplicationGroupPolicy.GetApplicationGroupPolicy() *ApplicationGroupPolicy`
- New function `*ClientFactory.NewApplicationGroupClient() *ApplicationGroupClient`
- New function `*ClientFactory.NewNetworkSecurityPerimeterConfigurationClient() *NetworkSecurityPerimeterConfigurationClient`
- New function `*ClientFactory.NewNetworkSecurityPerimeterConfigurationsClient() *NetworkSecurityPerimeterConfigurationsClient`
- New function `*ThrottlingPolicy.GetApplicationGroupPolicy() *ApplicationGroupPolicy`
- New function `NewNetworkSecurityPerimeterConfigurationClient(string, azcore.TokenCredential, *arm.ClientOptions) (*NetworkSecurityPerimeterConfigurationClient, error)`
- New function `*NetworkSecurityPerimeterConfigurationClient.List(context.Context, string, string, *NetworkSecurityPerimeterConfigurationClientListOptions) (NetworkSecurityPerimeterConfigurationClientListResponse, error)`
- New function `NewNetworkSecurityPerimeterConfigurationsClient(string, azcore.TokenCredential, *arm.ClientOptions) (*NetworkSecurityPerimeterConfigurationsClient, error)`
- New function `*NetworkSecurityPerimeterConfigurationsClient.BeginCreateOrUpdate(context.Context, string, string, string, *NetworkSecurityPerimeterConfigurationsClientBeginCreateOrUpdateOptions) (*runtime.Poller[NetworkSecurityPerimeterConfigurationsClientCreateOrUpdateResponse], error)`
- New struct `ApplicationGroup`
- New struct `ApplicationGroupListResult`
- New struct `ApplicationGroupProperties`
- New struct `CaptureIdentity`
- New struct `NetworkSecurityPerimeter`
- New struct `NetworkSecurityPerimeterConfiguration`
- New struct `NetworkSecurityPerimeterConfigurationList`
- New struct `NetworkSecurityPerimeterConfigurationProperties`
- New struct `NetworkSecurityPerimeterConfigurationPropertiesProfile`
- New struct `NetworkSecurityPerimeterConfigurationPropertiesResourceAssociation`
- New struct `NspAccessRule`
- New struct `NspAccessRuleProperties`
- New struct `NspAccessRulePropertiesSubscriptionsItem`
- New struct `ProvisioningIssue`
- New struct `ProvisioningIssueProperties`
- New struct `RetentionDescription`
- New struct `ThrottlingPolicy`
- New field `ProvisioningState`, `SupportsScaling` in struct `ClusterProperties`
- New field `Identity` in struct `Destination`
- New field `MinimumTLSVersion`, `PublicNetworkAccess` in struct `EHNamespaceProperties`
- New field `RetentionDescription`, `UserMetadata` in struct `Properties`


## 1.3.0-beta.1 (2023-11-30)
### Features Added

- New value `PublicNetworkAccessFlagSecuredByPerimeter` added to enum type `PublicNetworkAccessFlag`
- New enum type `ApplicationGroupPolicyType` with values `ApplicationGroupPolicyTypeThrottlingPolicy`
- New enum type `CleanupPolicyRetentionDescription` with values `CleanupPolicyRetentionDescriptionCompaction`, `CleanupPolicyRetentionDescriptionDelete`
- New enum type `MetricID` with values `MetricIDIncomingBytes`, `MetricIDIncomingMessages`, `MetricIDOutgoingBytes`, `MetricIDOutgoingMessages`
- New enum type `NetworkSecurityPerimeterConfigurationProvisioningState` with values `NetworkSecurityPerimeterConfigurationProvisioningStateAccepted`, `NetworkSecurityPerimeterConfigurationProvisioningStateCanceled`, `NetworkSecurityPerimeterConfigurationProvisioningStateCreating`, `NetworkSecurityPerimeterConfigurationProvisioningStateDeleted`, `NetworkSecurityPerimeterConfigurationProvisioningStateDeleting`, `NetworkSecurityPerimeterConfigurationProvisioningStateFailed`, `NetworkSecurityPerimeterConfigurationProvisioningStateInvalidResponse`, `NetworkSecurityPerimeterConfigurationProvisioningStateSucceeded`, `NetworkSecurityPerimeterConfigurationProvisioningStateSucceededWithIssues`, `NetworkSecurityPerimeterConfigurationProvisioningStateUnknown`, `NetworkSecurityPerimeterConfigurationProvisioningStateUpdating`
- New enum type `NspAccessRuleDirection` with values `NspAccessRuleDirectionInbound`, `NspAccessRuleDirectionOutbound`
- New enum type `PublicNetworkAccess` with values `PublicNetworkAccessDisabled`, `PublicNetworkAccessEnabled`, `PublicNetworkAccessSecuredByPerimeter`
- New enum type `ResourceAssociationAccessMode` with values `ResourceAssociationAccessModeAuditMode`, `ResourceAssociationAccessModeEnforcedMode`, `ResourceAssociationAccessModeLearningMode`, `ResourceAssociationAccessModeNoAssociationMode`, `ResourceAssociationAccessModeUnspecifiedMode`
- New enum type `TLSVersion` with values `TLSVersionOne0`, `TLSVersionOne1`, `TLSVersionOne2`
- New function `NewApplicationGroupClient(string, azcore.TokenCredential, *arm.ClientOptions) (*ApplicationGroupClient, error)`
- New function `*ApplicationGroupClient.CreateOrUpdateApplicationGroup(context.Context, string, string, string, ApplicationGroup, *ApplicationGroupClientCreateOrUpdateApplicationGroupOptions) (ApplicationGroupClientCreateOrUpdateApplicationGroupResponse, error)`
- New function `*ApplicationGroupClient.Delete(context.Context, string, string, string, *ApplicationGroupClientDeleteOptions) (ApplicationGroupClientDeleteResponse, error)`
- New function `*ApplicationGroupClient.Get(context.Context, string, string, string, *ApplicationGroupClientGetOptions) (ApplicationGroupClientGetResponse, error)`
- New function `*ApplicationGroupClient.NewListByNamespacePager(string, string, *ApplicationGroupClientListByNamespaceOptions) *runtime.Pager[ApplicationGroupClientListByNamespaceResponse]`
- New function `*ApplicationGroupPolicy.GetApplicationGroupPolicy() *ApplicationGroupPolicy`
- New function `*ClientFactory.NewApplicationGroupClient() *ApplicationGroupClient`
- New function `*ClientFactory.NewNetworkSecurityPerimeterConfigurationClient() *NetworkSecurityPerimeterConfigurationClient`
- New function `*ClientFactory.NewNetworkSecurityPerimeterConfigurationsClient() *NetworkSecurityPerimeterConfigurationsClient`
- New function `*ThrottlingPolicy.GetApplicationGroupPolicy() *ApplicationGroupPolicy`
- New function `NewNetworkSecurityPerimeterConfigurationClient(string, azcore.TokenCredential, *arm.ClientOptions) (*NetworkSecurityPerimeterConfigurationClient, error)`
- New function `*NetworkSecurityPerimeterConfigurationClient.List(context.Context, string, string, *NetworkSecurityPerimeterConfigurationClientListOptions) (NetworkSecurityPerimeterConfigurationClientListResponse, error)`
- New function `NewNetworkSecurityPerimeterConfigurationsClient(string, azcore.TokenCredential, *arm.ClientOptions) (*NetworkSecurityPerimeterConfigurationsClient, error)`
- New function `*NetworkSecurityPerimeterConfigurationsClient.BeginCreateOrUpdate(context.Context, string, string, string, *NetworkSecurityPerimeterConfigurationsClientBeginCreateOrUpdateOptions) (*runtime.Poller[NetworkSecurityPerimeterConfigurationsClientCreateOrUpdateResponse], error)`
- New struct `ApplicationGroup`
- New struct `ApplicationGroupListResult`
- New struct `ApplicationGroupProperties`
- New struct `NetworkSecurityPerimeter`
- New struct `NetworkSecurityPerimeterConfiguration`
- New struct `NetworkSecurityPerimeterConfigurationList`
- New struct `NetworkSecurityPerimeterConfigurationProperties`
- New struct `NetworkSecurityPerimeterConfigurationPropertiesProfile`
- New struct `NetworkSecurityPerimeterConfigurationPropertiesResourceAssociation`
- New struct `NspAccessRule`
- New struct `NspAccessRuleProperties`
- New struct `NspAccessRulePropertiesSubscriptionsItem`
- New struct `ProvisioningIssue`
- New struct `ProvisioningIssueProperties`
- New struct `RetentionDescription`
- New struct `ThrottlingPolicy`
- New field `SupportsScaling` in struct `ClusterProperties`
- New field `MinimumTLSVersion`, `PublicNetworkAccess` in struct `EHNamespaceProperties`
- New field `RetentionDescription` in struct `Properties`


## 1.2.0 (2023-11-24)
### Features Added

- Support for test fakes and OpenTelemetry trace spans.


## 1.2.0-beta.1 (2023-04-28)
### Features Added

- New value `PublicNetworkAccessFlagSecuredByPerimeter` added to enum type `PublicNetworkAccessFlag`
- New enum type `ApplicationGroupPolicyType` with values `ApplicationGroupPolicyTypeThrottlingPolicy`
- New enum type `CleanupPolicyRetentionDescription` with values `CleanupPolicyRetentionDescriptionCompaction`, `CleanupPolicyRetentionDescriptionDelete`
- New enum type `MetricID` with values `MetricIDIncomingBytes`, `MetricIDIncomingMessages`, `MetricIDOutgoingBytes`, `MetricIDOutgoingMessages`
- New enum type `NetworkSecurityPerimeterConfigurationProvisioningState` with values `NetworkSecurityPerimeterConfigurationProvisioningStateAccepted`, `NetworkSecurityPerimeterConfigurationProvisioningStateCanceled`, `NetworkSecurityPerimeterConfigurationProvisioningStateCreating`, `NetworkSecurityPerimeterConfigurationProvisioningStateDeleted`, `NetworkSecurityPerimeterConfigurationProvisioningStateDeleting`, `NetworkSecurityPerimeterConfigurationProvisioningStateFailed`, `NetworkSecurityPerimeterConfigurationProvisioningStateInvalidResponse`, `NetworkSecurityPerimeterConfigurationProvisioningStateSucceeded`, `NetworkSecurityPerimeterConfigurationProvisioningStateSucceededWithIssues`, `NetworkSecurityPerimeterConfigurationProvisioningStateUnknown`, `NetworkSecurityPerimeterConfigurationProvisioningStateUpdating`
- New enum type `NspAccessRuleDirection` with values `NspAccessRuleDirectionInbound`, `NspAccessRuleDirectionOutbound`
- New enum type `PublicNetworkAccess` with values `PublicNetworkAccessDisabled`, `PublicNetworkAccessEnabled`, `PublicNetworkAccessSecuredByPerimeter`
- New enum type `ResourceAssociationAccessMode` with values `ResourceAssociationAccessModeAuditMode`, `ResourceAssociationAccessModeEnforcedMode`, `ResourceAssociationAccessModeLearningMode`, `ResourceAssociationAccessModeNoAssociationMode`, `ResourceAssociationAccessModeUnspecifiedMode`
- New enum type `TLSVersion` with values `TLSVersionOne0`, `TLSVersionOne1`, `TLSVersionOne2`
- New function `NewApplicationGroupClient(string, azcore.TokenCredential, *arm.ClientOptions) (*ApplicationGroupClient, error)`
- New function `*ApplicationGroupClient.CreateOrUpdateApplicationGroup(context.Context, string, string, string, ApplicationGroup, *ApplicationGroupClientCreateOrUpdateApplicationGroupOptions) (ApplicationGroupClientCreateOrUpdateApplicationGroupResponse, error)`
- New function `*ApplicationGroupClient.Delete(context.Context, string, string, string, *ApplicationGroupClientDeleteOptions) (ApplicationGroupClientDeleteResponse, error)`
- New function `*ApplicationGroupClient.Get(context.Context, string, string, string, *ApplicationGroupClientGetOptions) (ApplicationGroupClientGetResponse, error)`
- New function `*ApplicationGroupClient.NewListByNamespacePager(string, string, *ApplicationGroupClientListByNamespaceOptions) *runtime.Pager[ApplicationGroupClientListByNamespaceResponse]`
- New function `*ApplicationGroupPolicy.GetApplicationGroupPolicy() *ApplicationGroupPolicy`
- New function `*ClientFactory.NewApplicationGroupClient() *ApplicationGroupClient`
- New function `*ClientFactory.NewNetworkSecurityPerimeterConfigurationClient() *NetworkSecurityPerimeterConfigurationClient`
- New function `*ClientFactory.NewNetworkSecurityPerimeterConfigurationsClient() *NetworkSecurityPerimeterConfigurationsClient`
- New function `*ThrottlingPolicy.GetApplicationGroupPolicy() *ApplicationGroupPolicy`
- New function `NewNetworkSecurityPerimeterConfigurationClient(string, azcore.TokenCredential, *arm.ClientOptions) (*NetworkSecurityPerimeterConfigurationClient, error)`
- New function `*NetworkSecurityPerimeterConfigurationClient.List(context.Context, string, string, *NetworkSecurityPerimeterConfigurationClientListOptions) (NetworkSecurityPerimeterConfigurationClientListResponse, error)`
- New function `NewNetworkSecurityPerimeterConfigurationsClient(string, azcore.TokenCredential, *arm.ClientOptions) (*NetworkSecurityPerimeterConfigurationsClient, error)`
- New function `*NetworkSecurityPerimeterConfigurationsClient.BeginCreateOrUpdate(context.Context, string, string, string, *NetworkSecurityPerimeterConfigurationsClientBeginCreateOrUpdateOptions) (*runtime.Poller[NetworkSecurityPerimeterConfigurationsClientCreateOrUpdateResponse], error)`
- New struct `ApplicationGroup`
- New struct `ApplicationGroupListResult`
- New struct `ApplicationGroupProperties`
- New struct `NetworkSecurityPerimeter`
- New struct `NetworkSecurityPerimeterConfiguration`
- New struct `NetworkSecurityPerimeterConfigurationList`
- New struct `NetworkSecurityPerimeterConfigurationProperties`
- New struct `NetworkSecurityPerimeterConfigurationPropertiesProfile`
- New struct `NetworkSecurityPerimeterConfigurationPropertiesResourceAssociation`
- New struct `NspAccessRule`
- New struct `NspAccessRuleProperties`
- New struct `NspAccessRulePropertiesSubscriptionsItem`
- New struct `ProvisioningIssue`
- New struct `ProvisioningIssueProperties`
- New struct `RetentionDescription`
- New struct `ThrottlingPolicy`
- New field `SupportsScaling` in struct `ClusterProperties`
- New field `MinimumTLSVersion` in struct `EHNamespaceProperties`
- New field `PublicNetworkAccess` in struct `EHNamespaceProperties`
- New field `RetentionDescription` in struct `Properties`


## 1.1.1 (2023-04-14)
### Bug Fixes

- Fix serialization bug of empty value of `any` type.


## 1.1.0 (2023-04-06)
### Features Added

- New struct `ClientFactory` which is a client factory used to create any client in this module


## 1.0.0 (2022-05-16)

The package of `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/eventhub/armeventhub` is using our [next generation design principles](https://azure.github.io/azure-sdk/general_introduction.html) since version 1.0.0, which contains breaking changes.

To migrate the existing applications to the latest version, please refer to [Migration Guide](https://aka.ms/azsdk/go/mgmt/migration).

To learn more, please refer to our documentation [Quick Start](https://aka.ms/azsdk/go/mgmt).
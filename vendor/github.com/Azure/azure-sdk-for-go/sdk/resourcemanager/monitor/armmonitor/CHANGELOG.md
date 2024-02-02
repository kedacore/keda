# Release History

## 0.11.0 (2023-11-24)
### Features Added

- Support for test fakes and OpenTelemetry trace spans.


## 0.10.2 (2023-10-09)

### Other Changes

- Updated to latest `azcore` beta.

## 0.10.1 (2023-07-19)

### Bug Fixes

- Fixed a potential panic in faked paged and long-running operations.

## 0.10.0 (2023-06-13)

### Features Added

- Support for test fakes and OpenTelemetry trace spans.

## 0.9.1 (2023-04-14)
### Bug Fixes

- Fix serialization bug of empty value of `any` type.


## 0.9.0 (2023-03-24)
### Breaking Changes

- Function `NewMetricDefinitionsClient` parameter(s) have been changed from `(azcore.TokenCredential, *arm.ClientOptions)` to `(string, azcore.TokenCredential, *arm.ClientOptions)`
- Function `NewMetricsClient` parameter(s) have been changed from `(azcore.TokenCredential, *arm.ClientOptions)` to `(string, azcore.TokenCredential, *arm.ClientOptions)`
- Type of `ErrorContract.Error` has been changed from `*ErrorResponseDetails` to `*ErrorResponse`
- Type of `Metric.Unit` has been changed from `*MetricUnit` to `*Unit`
- Function `*ActionGroupsClient.BeginCreateNotificationsAtResourceGroupLevel` has been removed
- Function `*ActionGroupsClient.GetTestNotifications` has been removed
- Function `*ActionGroupsClient.GetTestNotificationsAtResourceGroupLevel` has been removed
- Function `*ActionGroupsClient.BeginPostTestNotifications` has been removed

### Features Added

- New struct `ClientFactory` which is a client factory used to create any client in this module
- New value `KnownDataCollectionEndpointProvisioningStateCanceled` added to enum type `KnownDataCollectionEndpointProvisioningState`
- New value `KnownDataCollectionRuleAssociationProvisioningStateCanceled` added to enum type `KnownDataCollectionRuleAssociationProvisioningState`
- New value `KnownDataCollectionRuleProvisioningStateCanceled` added to enum type `KnownDataCollectionRuleProvisioningState`
- New value `KnownPublicNetworkAccessOptionsSecuredByPerimeter` added to enum type `KnownPublicNetworkAccessOptions`
- New enum type `ActionType` with values `ActionTypeInternal`
- New enum type `IdentityType` with values `IdentityTypeNone`, `IdentityTypeSystemAssigned`, `IdentityTypeUserAssigned`
- New enum type `KnownLocationSpecProvisioningStatus` with values `KnownLocationSpecProvisioningStatusCanceled`, `KnownLocationSpecProvisioningStatusCreating`, `KnownLocationSpecProvisioningStatusDeleting`, `KnownLocationSpecProvisioningStatusFailed`, `KnownLocationSpecProvisioningStatusSucceeded`, `KnownLocationSpecProvisioningStatusUpdating`
- New enum type `KnownPrometheusForwarderDataSourceStreams` with values `KnownPrometheusForwarderDataSourceStreamsMicrosoftPrometheusMetrics`
- New enum type `ManagedServiceIdentityType` with values `ManagedServiceIdentityTypeNone`, `ManagedServiceIdentityTypeSystemAssigned`, `ManagedServiceIdentityTypeSystemAssignedUserAssigned`, `ManagedServiceIdentityTypeUserAssigned`
- New enum type `MetricAggregationType` with values `MetricAggregationTypeAverage`, `MetricAggregationTypeCount`, `MetricAggregationTypeMaximum`, `MetricAggregationTypeMinimum`, `MetricAggregationTypeNone`, `MetricAggregationTypeTotal`
- New enum type `MetricResultType` with values `MetricResultTypeData`, `MetricResultTypeMetadata`
- New enum type `Origin` with values `OriginSystem`, `OriginUser`, `OriginUserSystem`
- New enum type `ProvisioningState` with values `ProvisioningStateCanceled`, `ProvisioningStateCreating`, `ProvisioningStateDeleting`, `ProvisioningStateFailed`, `ProvisioningStateSucceeded`
- New enum type `PublicNetworkAccess` with values `PublicNetworkAccessDisabled`, `PublicNetworkAccessEnabled`, `PublicNetworkAccessSecuredByPerimeter`
- New enum type `Unit` with values `UnitBitsPerSecond`, `UnitByteSeconds`, `UnitBytes`, `UnitBytesPerSecond`, `UnitCores`, `UnitCount`, `UnitCountPerSecond`, `UnitMilliCores`, `UnitMilliSeconds`, `UnitNanoCores`, `UnitPercent`, `UnitSeconds`, `UnitUnspecified`
- New function `NewAzureMonitorWorkspacesClient(string, azcore.TokenCredential, *arm.ClientOptions) (*AzureMonitorWorkspacesClient, error)`
- New function `*AzureMonitorWorkspacesClient.Create(context.Context, string, string, AzureMonitorWorkspaceResource, *AzureMonitorWorkspacesClientCreateOptions) (AzureMonitorWorkspacesClientCreateResponse, error)`
- New function `*AzureMonitorWorkspacesClient.Delete(context.Context, string, string, *AzureMonitorWorkspacesClientDeleteOptions) (AzureMonitorWorkspacesClientDeleteResponse, error)`
- New function `*AzureMonitorWorkspacesClient.Get(context.Context, string, string, *AzureMonitorWorkspacesClientGetOptions) (AzureMonitorWorkspacesClientGetResponse, error)`
- New function `*AzureMonitorWorkspacesClient.NewListByResourceGroupPager(string, *AzureMonitorWorkspacesClientListByResourceGroupOptions) *runtime.Pager[AzureMonitorWorkspacesClientListByResourceGroupResponse]`
- New function `*AzureMonitorWorkspacesClient.NewListBySubscriptionPager(*AzureMonitorWorkspacesClientListBySubscriptionOptions) *runtime.Pager[AzureMonitorWorkspacesClientListBySubscriptionResponse]`
- New function `*AzureMonitorWorkspacesClient.Update(context.Context, string, string, *AzureMonitorWorkspacesClientUpdateOptions) (AzureMonitorWorkspacesClientUpdateResponse, error)`
- New function `*MetricDefinitionsClient.NewListAtSubscriptionScopePager(string, *MetricDefinitionsClientListAtSubscriptionScopeOptions) *runtime.Pager[MetricDefinitionsClientListAtSubscriptionScopeResponse]`
- New function `*MetricsClient.ListAtSubscriptionScope(context.Context, string, *MetricsClientListAtSubscriptionScopeOptions) (MetricsClientListAtSubscriptionScopeResponse, error)`
- New function `*MetricsClient.ListAtSubscriptionScopePost(context.Context, string, *MetricsClientListAtSubscriptionScopePostOptions) (MetricsClientListAtSubscriptionScopePostResponse, error)`
- New function `NewOperationsForMonitorClient(azcore.TokenCredential, *arm.ClientOptions) (*OperationsForMonitorClient, error)`
- New function `*OperationsForMonitorClient.NewListPager(*OperationsForMonitorClientListOptions) *runtime.Pager[OperationsForMonitorClientListResponse]`
- New function `NewTenantActionGroupsClient(azcore.TokenCredential, *arm.ClientOptions) (*TenantActionGroupsClient, error)`
- New function `*TenantActionGroupsClient.CreateOrUpdate(context.Context, string, string, string, TenantActionGroupResource, *TenantActionGroupsClientCreateOrUpdateOptions) (TenantActionGroupsClientCreateOrUpdateResponse, error)`
- New function `*TenantActionGroupsClient.Delete(context.Context, string, string, string, *TenantActionGroupsClientDeleteOptions) (TenantActionGroupsClientDeleteResponse, error)`
- New function `*TenantActionGroupsClient.Get(context.Context, string, string, string, *TenantActionGroupsClientGetOptions) (TenantActionGroupsClientGetResponse, error)`
- New function `*TenantActionGroupsClient.NewListByManagementGroupIDPager(string, string, *TenantActionGroupsClientListByManagementGroupIDOptions) *runtime.Pager[TenantActionGroupsClientListByManagementGroupIDResponse]`
- New function `*TenantActionGroupsClient.Update(context.Context, string, string, string, ActionGroupPatchBodyAutoGenerated, *TenantActionGroupsClientUpdateOptions) (TenantActionGroupsClientUpdateResponse, error)`
- New struct `ActionGroupPatchAutoGenerated`
- New struct `ActionGroupPatchBodyAutoGenerated`
- New struct `AzureAppPushReceiverAutoGenerated`
- New struct `AzureMonitorWorkspace`
- New struct `AzureMonitorWorkspaceDefaultIngestionSettings`
- New struct `AzureMonitorWorkspaceMetrics`
- New struct `AzureMonitorWorkspaceResource`
- New struct `AzureMonitorWorkspaceResourceForUpdate`
- New struct `AzureMonitorWorkspaceResourceListResult`
- New struct `AzureMonitorWorkspaceResourceProperties`
- New struct `DataCollectionEndpointFailoverConfiguration`
- New struct `DataCollectionEndpointMetadata`
- New struct `DataCollectionEndpointMetricsIngestion`
- New struct `DataCollectionEndpointResourceIdentity`
- New struct `DataCollectionRuleResourceIdentity`
- New struct `DataImportSources`
- New struct `DataImportSourcesEventHub`
- New struct `DataSourcesSpecDataImports`
- New struct `EmailReceiverAutoGenerated`
- New struct `ErrorContractAutoGenerated`
- New struct `ErrorDetailAutoGenerated`
- New struct `ErrorResponseAutoGenerated2`
- New struct `EventHubDataSource`
- New struct `EventHubDestination`
- New struct `EventHubDirectDestination`
- New struct `FailoverConfigurationSpec`
- New struct `Identity`
- New struct `IngestionSettings`
- New struct `LocationSpec`
- New struct `ManagedServiceIdentity`
- New struct `Metrics`
- New struct `MetricsIngestionEndpointSpec`
- New struct `MonitoringAccountDestination`
- New struct `OperationAutoGenerated`
- New struct `OperationDisplayAutoGenerated`
- New struct `OperationListResultAutoGenerated`
- New struct `PlatformTelemetryDataSource`
- New struct `PrivateLinkScopedResource`
- New struct `PrometheusForwarderDataSource`
- New struct `ResourceAutoGenerated5`
- New struct `ResourceForUpdateIdentity`
- New struct `RuleResolveConfiguration`
- New struct `SmsReceiverAutoGenerated`
- New struct `StorageBlobDestination`
- New struct `StorageTableDestination`
- New struct `SubscriptionScopeMetric`
- New struct `SubscriptionScopeMetricDefinition`
- New struct `SubscriptionScopeMetricDefinitionCollection`
- New struct `SubscriptionScopeMetricResponse`
- New struct `SubscriptionScopeMetricsRequestBodyParameters`
- New struct `TenantActionGroup`
- New struct `TenantActionGroupList`
- New struct `TenantActionGroupResource`
- New struct `TrackedResourceAutoGenerated`
- New struct `UserAssignedIdentity`
- New struct `UserIdentityProperties`
- New struct `VoiceReceiverAutoGenerated`
- New struct `WebhookReceiverAutoGenerated`
- New struct `WindowsFirewallLogsDataSource`
- New field `FailoverConfiguration` in struct `DataCollectionEndpoint`
- New field `Metadata` in struct `DataCollectionEndpoint`
- New field `MetricsIngestion` in struct `DataCollectionEndpoint`
- New field `PrivateLinkScopedResources` in struct `DataCollectionEndpoint`
- New field `Identity` in struct `DataCollectionEndpointResource`
- New field `FailoverConfiguration` in struct `DataCollectionEndpointResourceProperties`
- New field `Metadata` in struct `DataCollectionEndpointResourceProperties`
- New field `MetricsIngestion` in struct `DataCollectionEndpointResourceProperties`
- New field `PrivateLinkScopedResources` in struct `DataCollectionEndpointResourceProperties`
- New field `ProvisionedByResourceID` in struct `DataCollectionRuleAssociationMetadata`
- New field `DataImports` in struct `DataCollectionRuleDataSources`
- New field `PlatformTelemetry` in struct `DataCollectionRuleDataSources`
- New field `PrometheusForwarder` in struct `DataCollectionRuleDataSources`
- New field `WindowsFirewallLogs` in struct `DataCollectionRuleDataSources`
- New field `EventHubs` in struct `DataCollectionRuleDestinations`
- New field `EventHubsDirect` in struct `DataCollectionRuleDestinations`
- New field `MonitoringAccounts` in struct `DataCollectionRuleDestinations`
- New field `StorageAccounts` in struct `DataCollectionRuleDestinations`
- New field `StorageBlobsDirect` in struct `DataCollectionRuleDestinations`
- New field `StorageTablesDirect` in struct `DataCollectionRuleDestinations`
- New field `ProvisionedByResourceID` in struct `DataCollectionRuleMetadata`
- New field `Identity` in struct `DataCollectionRuleResource`
- New field `BuiltInTransform` in struct `DataFlow`
- New field `DataImports` in struct `DataSourcesSpec`
- New field `PlatformTelemetry` in struct `DataSourcesSpec`
- New field `PrometheusForwarder` in struct `DataSourcesSpec`
- New field `WindowsFirewallLogs` in struct `DataSourcesSpec`
- New field `EventHubs` in struct `DestinationsSpec`
- New field `EventHubsDirect` in struct `DestinationsSpec`
- New field `MonitoringAccounts` in struct `DestinationsSpec`
- New field `StorageAccounts` in struct `DestinationsSpec`
- New field `StorageBlobsDirect` in struct `DestinationsSpec`
- New field `StorageTablesDirect` in struct `DestinationsSpec`
- New field `ProvisionedByResourceID` in struct `Metadata`
- New field `AutoAdjustTimegrain` in struct `MetricsClientListOptions`
- New field `ValidateDimensions` in struct `MetricsClientListOptions`
- New field `Identity` in struct `ResourceForUpdate`
- New field `PublicNetworkAccess` in struct `ScheduledQueryRuleProperties`
- New field `RuleResolveConfiguration` in struct `ScheduledQueryRuleProperties`
- New field `Identity` in struct `ScheduledQueryRuleResource`
- New field `Identity` in struct `ScheduledQueryRuleResourcePatch`


## 0.8.0 (2022-10-18)
### Breaking Changes

- Type alias `AlertSeverity` type has been changed from `string` to `int64`
- Function `*ScheduledQueryRulesClient.CreateOrUpdate` parameter(s) have been changed from `(context.Context, string, string, LogSearchRuleResource, *ScheduledQueryRulesClientCreateOrUpdateOptions)` to `(context.Context, string, string, ScheduledQueryRuleResource, *ScheduledQueryRulesClientCreateOrUpdateOptions)`
- Function `*ScheduledQueryRulesClient.Update` parameter(s) have been changed from `(context.Context, string, string, LogSearchRuleResourcePatch, *ScheduledQueryRulesClientUpdateOptions)` to `(context.Context, string, string, ScheduledQueryRuleResourcePatch, *ScheduledQueryRulesClientUpdateOptions)`
- Type of `OperationStatus.Error` has been changed from `*ErrorResponseCommon` to `*ErrorDetail`
- Type of `PrivateEndpointConnectionProperties.PrivateLinkServiceConnectionState` has been changed from `*PrivateLinkServiceConnectionStateProperty` to `*PrivateLinkServiceConnectionState`
- Type of `PrivateEndpointConnectionProperties.PrivateEndpoint` has been changed from `*PrivateEndpointProperty` to `*PrivateEndpoint`
- Type of `PrivateEndpointConnectionProperties.ProvisioningState` has been changed from `*string` to `*PrivateEndpointConnectionProvisioningState`
- Type of `ErrorContract.Error` has been changed from `*ErrorResponse` to `*ErrorResponseDetails`
- Type of `Dimension.Operator` has been changed from `*Operator` to `*DimensionOperator`
- Type alias `ConditionalOperator`, const `ConditionalOperatorLessThanOrEqual`, `ConditionalOperatorEqual`, `ConditionalOperatorGreaterThanOrEqual`, `ConditionalOperatorLessThan`, `ConditionalOperatorGreaterThan` and function `PossibleConditionalOperatorValues` have been removed
- Type alias `Enabled`, const `EnabledTrue`, `EnabledFalse` and function `PossibleEnabledValues` have been removed
- Type alias `QueryType`, const `QueryTypeResultCount` and function `PossibleQueryTypeValues` have been removed
- Type alias `MetricTriggerType`, const `MetricTriggerTypeConsecutive`, `MetricTriggerTypeTotal` and function `PossibleMetricTriggerTypeValues` have been removed
- Type alias `ProvisioningState`, const `ProvisioningStateSucceeded`, `ProvisioningStateFailed`, `ProvisioningStateDeploying`, `ProvisioningStateCanceled` and function `PossibleProvisioningStateValues` have been removed
- Const `OperatorInclude` has been removed
- Function `*DiagnosticSettingsClient.List` has been changed to `*DiagnosticSettingsClient.NewListPager(string, *DiagnosticSettingsClientListOptions) *runtime.Pager[DiagnosticSettingsClientListResponse]`
- Function `*PrivateEndpointConnectionsClient.NewListByPrivateLinkScopePager` has been changed to `*PrivateEndpointConnectionsClient.ListByPrivateLinkScope(context.Context, string, string, *PrivateEndpointConnectionsClientListByPrivateLinkScopeOptions) (PrivateEndpointConnectionsClientListByPrivateLinkScopeResponse, error)`
- Function `*DiagnosticSettingsCategoryClient.List` has been changed to `*DiagnosticSettingsCategoryClient.NewListPager(string, *DiagnosticSettingsCategoryClientListOptions) *runtime.Pager[DiagnosticSettingsCategoryClientListResponse]`
- Function `*PrivateLinkResourcesClient.NewListByPrivateLinkScopePager` has been changed to `*PrivateLinkResourcesClient.ListByPrivateLinkScope(context.Context, string, string, *PrivateLinkResourcesClientListByPrivateLinkScopeOptions) (PrivateLinkResourcesClientListByPrivateLinkScopeResponse, error)`
- Struct `Action` and function `*Action.GetAction` have been removed
- Struct `AlertingAction` and function `*AlertingAction.GetAction` have been removed
- Struct `LogToMetricAction` and function `*LogToMetricAction.GetAction` have been removed
- Struct `AzNsActionGroup` has been removed
- Struct `Criteria` has been removed
- Struct `LogMetricTrigger` has been removed
- Struct `LogSearchRule` has been removed
- Struct `LogSearchRulePatch` has been removed
- Struct `LogSearchRuleResource` has been removed
- Struct `LogSearchRuleResourceCollection` has been removed
- Struct `LogSearchRuleResourcePatch` has been removed
- Struct `PrivateEndpointProperty` has been removed
- Struct `PrivateLinkScopesResource` has been removed
- Struct `PrivateLinkServiceConnectionStateProperty` has been removed
- Struct `Schedule` has been removed
- Struct `Source` has been removed
- Struct `TriggerCondition` has been removed
- Field `Filter` of struct `ScheduledQueryRulesClientListByResourceGroupOptions` has been removed
- Field `LogSearchRuleResource` of struct `ScheduledQueryRulesClientUpdateResponse` has been removed
- Field `LogSearchRuleResourceCollection` of struct `ScheduledQueryRulesClientListBySubscriptionResponse` has been removed
- Field `LogSearchRuleResourceCollection` of struct `ScheduledQueryRulesClientListByResourceGroupResponse` has been removed
- Field `NextLink` of struct `PrivateLinkResourceListResult` has been removed
- Field `Filter` of struct `ScheduledQueryRulesClientListBySubscriptionOptions` has been removed
- Field `LogSearchRuleResource` of struct `ScheduledQueryRulesClientGetResponse` has been removed
- Field `NextLink` of struct `PrivateEndpointConnectionListResult` has been removed
- Field `Identity` of struct `ActionGroupResource` has been removed
- Field `Kind` of struct `ActionGroupResource` has been removed
- Field `Identity` of struct `AzureResource` has been removed
- Field `Kind` of struct `AzureResource` has been removed
- Field `LogSearchRuleResource` of struct `ScheduledQueryRulesClientCreateOrUpdateResponse` has been removed

### Features Added

- New const `ConditionOperatorEquals`
- New const `KindLogAlert`
- New const `PrivateEndpointServiceConnectionStatusPending`
- New const `AccessModePrivateOnly`
- New const `PredictiveAutoscalePolicyScaleModeEnabled`
- New const `TimeAggregationTotal`
- New const `TimeAggregationAverage`
- New const `AccessModeOpen`
- New const `PredictiveAutoscalePolicyScaleModeForecastOnly`
- New const `PrivateEndpointConnectionProvisioningStateFailed`
- New const `PredictiveAutoscalePolicyScaleModeDisabled`
- New const `PrivateEndpointConnectionProvisioningStateCreating`
- New const `TimeAggregationMinimum`
- New const `DimensionOperatorExclude`
- New const `DimensionOperatorInclude`
- New const `PrivateEndpointConnectionProvisioningStateDeleting`
- New const `PrivateEndpointServiceConnectionStatusRejected`
- New const `PrivateEndpointConnectionProvisioningStateSucceeded`
- New const `PrivateEndpointServiceConnectionStatusApproved`
- New const `KindLogToMetric`
- New const `TimeAggregationCount`
- New const `TimeAggregationMaximum`
- New type alias `PrivateEndpointServiceConnectionStatus`
- New type alias `DimensionOperator`
- New type alias `PredictiveAutoscalePolicyScaleMode`
- New type alias `AccessMode`
- New type alias `Kind`
- New type alias `TimeAggregation`
- New type alias `PrivateEndpointConnectionProvisioningState`
- New function `PossiblePrivateEndpointServiceConnectionStatusValues() []PrivateEndpointServiceConnectionStatus`
- New function `PossiblePredictiveAutoscalePolicyScaleModeValues() []PredictiveAutoscalePolicyScaleMode`
- New function `PossiblePrivateEndpointConnectionProvisioningStateValues() []PrivateEndpointConnectionProvisioningState`
- New function `NewPredictiveMetricClient(string, azcore.TokenCredential, *arm.ClientOptions) (*PredictiveMetricClient, error)`
- New function `PossibleTimeAggregationValues() []TimeAggregation`
- New function `PossibleAccessModeValues() []AccessMode`
- New function `*PredictiveMetricClient.Get(context.Context, string, string, string, string, string, string, string, *PredictiveMetricClientGetOptions) (PredictiveMetricClientGetResponse, error)`
- New function `PossibleKindValues() []Kind`
- New function `PossibleDimensionOperatorValues() []DimensionOperator`
- New struct `AccessModeSettings`
- New struct `AccessModeSettingsExclusion`
- New struct `Actions`
- New struct `AutoscaleErrorResponse`
- New struct `AutoscaleErrorResponseError`
- New struct `Condition`
- New struct `ConditionFailingPeriods`
- New struct `DefaultErrorResponse`
- New struct `ErrorResponseAdditionalInfo`
- New struct `ErrorResponseDetails`
- New struct `PredictiveAutoscalePolicy`
- New struct `PredictiveMetricClient`
- New struct `PredictiveMetricClientGetOptions`
- New struct `PredictiveMetricClientGetResponse`
- New struct `PredictiveResponse`
- New struct `PredictiveValue`
- New struct `PrivateEndpoint`
- New struct `PrivateLinkServiceConnectionState`
- New struct `ProxyResourceAutoGenerated`
- New struct `ResourceAutoGenerated`
- New struct `ResourceAutoGenerated2`
- New struct `ResourceAutoGenerated3`
- New struct `ResourceAutoGenerated4`
- New struct `ScheduledQueryRuleCriteria`
- New struct `ScheduledQueryRuleProperties`
- New struct `ScheduledQueryRuleResource`
- New struct `ScheduledQueryRuleResourceCollection`
- New struct `ScheduledQueryRuleResourcePatch`
- New struct `TrackedResource`
- New anonymous field `TestNotificationDetailsResponse` in struct `ActionGroupsClientCreateNotificationsAtActionGroupResourceLevelResponse`
- New field `SystemData` in struct `DiagnosticSettingsCategoryResource`
- New anonymous field `ScheduledQueryRuleResource` in struct `ScheduledQueryRulesClientCreateOrUpdateResponse`
- New anonymous field `TestNotificationDetailsResponse` in struct `ActionGroupsClientCreateNotificationsAtResourceGroupLevelResponse`
- New field `PredictiveAutoscalePolicy` in struct `AutoscaleSetting`
- New field `SystemData` in struct `Resource`
- New field `CategoryGroups` in struct `DiagnosticSettingsCategory`
- New field `SystemData` in struct `AzureMonitorPrivateLinkScope`
- New field `RequiredZoneNames` in struct `PrivateLinkResourceProperties`
- New anonymous field `ScheduledQueryRuleResource` in struct `ScheduledQueryRulesClientGetResponse`
- New field `MarketplacePartnerID` in struct `DiagnosticSettings`
- New anonymous field `ScheduledQueryRuleResourceCollection` in struct `ScheduledQueryRulesClientListByResourceGroupResponse`
- New field `SystemData` in struct `ScopedResource`
- New field `SystemData` in struct `DiagnosticSettingsResource`
- New field `CategoryGroup` in struct `LogSettings`
- New field `AccessModeSettings` in struct `AzureMonitorPrivateLinkScopeProperties`
- New anonymous field `ScheduledQueryRuleResource` in struct `ScheduledQueryRulesClientUpdateResponse`
- New field `SystemData` in struct `AutoscaleSettingResource`
- New anonymous field `TestNotificationDetailsResponse` in struct `ActionGroupsClientPostTestNotificationsResponse`
- New anonymous field `ScheduledQueryRuleResourceCollection` in struct `ScheduledQueryRulesClientListBySubscriptionResponse`


## 0.7.0 (2022-05-17)

The package of `github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/monitor/armmonitor` is using our [next generation design principles](https://azure.github.io/azure-sdk/general_introduction.html) since version 0.7.0, which contains breaking changes.

To migrate the existing applications to the latest version, please refer to [Migration Guide](https://aka.ms/azsdk/go/mgmt/migration).

To learn more, please refer to our documentation [Quick Start](https://aka.ms/azsdk/go/mgmt).

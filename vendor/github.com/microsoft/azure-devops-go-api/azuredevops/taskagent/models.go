// --------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.
// --------------------------------------------------------------------------------------------
// Generated file, DO NOT EDIT
// Changes may cause incorrect behavior and will be lost if the code is regenerated.
// --------------------------------------------------------------------------------------------

package taskagent

import (
	"github.com/google/uuid"
	"github.com/microsoft/azure-devops-go-api/azuredevops"
	"github.com/microsoft/azure-devops-go-api/azuredevops/distributedtaskcommon"
	"github.com/microsoft/azure-devops-go-api/azuredevops/forminput"
	"github.com/microsoft/azure-devops-go-api/azuredevops/webapi"
)

type AadLoginPromptOption string

type aadLoginPromptOptionValuesType struct {
	NoOption          AadLoginPromptOption
	Login             AadLoginPromptOption
	SelectAccount     AadLoginPromptOption
	FreshLogin        AadLoginPromptOption
	FreshLoginWithMfa AadLoginPromptOption
}

var AadLoginPromptOptionValues = aadLoginPromptOptionValuesType{
	// Do not provide a prompt option
	NoOption: "noOption",
	// Force the user to login again.
	Login: "login",
	// Force the user to select which account they are logging in with instead of automatically picking the user up from the session state. NOTE: This does not work for switching between the variants of a dual-homed user.
	SelectAccount: "selectAccount",
	// Force the user to login again. <remarks> Ignore current authentication state and force the user to authenticate again. This option should be used instead of Login. </remarks>
	FreshLogin: "freshLogin",
	// Force the user to login again with mfa. <remarks> Ignore current authentication state and force the user to authenticate again. This option should be used instead of Login, if MFA is required. </remarks>
	FreshLoginWithMfa: "freshLoginWithMfa",
}

type AadOauthTokenRequest struct {
	Refresh  *bool   `json:"refresh,omitempty"`
	Resource *string `json:"resource,omitempty"`
	TenantId *string `json:"tenantId,omitempty"`
	Token    *string `json:"token,omitempty"`
}

type AadOauthTokenResult struct {
	AccessToken       *string `json:"accessToken,omitempty"`
	RefreshTokenCache *string `json:"refreshTokenCache,omitempty"`
}

type AgentChangeEvent struct {
	Agent     *TaskAgent              `json:"agent,omitempty"`
	EventType *string                 `json:"eventType,omitempty"`
	Pool      *TaskAgentPoolReference `json:"pool,omitempty"`
	// Deprecated:
	PoolId *int `json:"poolId,omitempty"`
	// Deprecated:
	TimeStamp *azuredevops.Time `json:"timeStamp,omitempty"`
}

type AgentJobRequestMessage struct {
	Environment *JobEnvironment                 `json:"environment,omitempty"`
	JobId       *uuid.UUID                      `json:"jobId,omitempty"`
	JobName     *string                         `json:"jobName,omitempty"`
	JobRefName  *string                         `json:"jobRefName,omitempty"`
	MessageType *string                         `json:"messageType,omitempty"`
	Plan        *TaskOrchestrationPlanReference `json:"plan,omitempty"`
	Timeline    *TimelineReference              `json:"timeline,omitempty"`
	LockedUntil *azuredevops.Time               `json:"lockedUntil,omitempty"`
	LockToken   *uuid.UUID                      `json:"lockToken,omitempty"`
	RequestId   *uint64                         `json:"requestId,omitempty"`
	Tasks       *[]TaskInstance                 `json:"tasks,omitempty"`
}

type AgentMigrationMessage struct {
	AccessToken *string `json:"accessToken,omitempty"`
}

type AgentPoolEvent struct {
	EventType *string        `json:"eventType,omitempty"`
	Pool      *TaskAgentPool `json:"pool,omitempty"`
}

type AgentQueueEvent struct {
	EventType *string         `json:"eventType,omitempty"`
	Queue     *TaskAgentQueue `json:"queue,omitempty"`
}

type AgentQueuesEvent struct {
	EventType *string           `json:"eventType,omitempty"`
	Queues    *[]TaskAgentQueue `json:"queues,omitempty"`
}

type AgentRefreshMessage struct {
	AgentId       *int        `json:"agentId,omitempty"`
	TargetVersion *string     `json:"targetVersion,omitempty"`
	Timeout       interface{} `json:"timeout,omitempty"`
}

type AuditAction string

type auditActionValuesType struct {
	Add      AuditAction
	Update   AuditAction
	Delete   AuditAction
	Undelete AuditAction
}

var AuditActionValues = auditActionValuesType{
	Add:      "add",
	Update:   "update",
	Delete:   "delete",
	Undelete: "undelete",
}

type AuthenticationSchemeReference struct {
	Inputs *map[string]string `json:"inputs,omitempty"`
	Type   *string            `json:"type,omitempty"`
}

type AuthorizationHeader struct {
	// Gets or sets the name of authorization header.
	Name *string `json:"name,omitempty"`
	// Gets or sets the value of authorization header.
	Value *string `json:"value,omitempty"`
}

type AzureKeyVaultPermission struct {
	Provisioned      *bool   `json:"provisioned,omitempty"`
	ResourceProvider *string `json:"resourceProvider,omitempty"`
	ResourceGroup    *string `json:"resourceGroup,omitempty"`
	Vault            *string `json:"vault,omitempty"`
}

type AzureKeyVaultVariableGroupProviderData struct {
	LastRefreshedOn   *azuredevops.Time `json:"lastRefreshedOn,omitempty"`
	ServiceEndpointId *uuid.UUID        `json:"serviceEndpointId,omitempty"`
	Vault             *string           `json:"vault,omitempty"`
}

type AzureKeyVaultVariableValue struct {
	IsSecret    *bool             `json:"isSecret,omitempty"`
	Value       *string           `json:"value,omitempty"`
	ContentType *string           `json:"contentType,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Expires     *azuredevops.Time `json:"expires,omitempty"`
}

// Azure Management Group
type AzureManagementGroup struct {
	// Display name of azure management group
	DisplayName *string `json:"displayName,omitempty"`
	// Id of azure management group
	Id *string `json:"id,omitempty"`
	// Azure management group name
	Name *string `json:"name,omitempty"`
	// Id of tenant from which azure management group belongs
	TenantId *string `json:"tenantId,omitempty"`
}

// Azure management group query result
type AzureManagementGroupQueryResult struct {
	// Error message in case of an exception
	ErrorMessage *string `json:"errorMessage,omitempty"`
	// List of azure management groups
	Value *[]AzureManagementGroup `json:"value,omitempty"`
}

type AzurePermission struct {
	Provisioned      *bool   `json:"provisioned,omitempty"`
	ResourceProvider *string `json:"resourceProvider,omitempty"`
}

type AzureResourcePermission struct {
	Provisioned      *bool   `json:"provisioned,omitempty"`
	ResourceProvider *string `json:"resourceProvider,omitempty"`
	ResourceGroup    *string `json:"resourceGroup,omitempty"`
}

type AzureRoleAssignmentPermission struct {
	Provisioned      *bool      `json:"provisioned,omitempty"`
	ResourceProvider *string    `json:"resourceProvider,omitempty"`
	RoleAssignmentId *uuid.UUID `json:"roleAssignmentId,omitempty"`
}

type AzureSpnOperationStatus struct {
	State         *string `json:"state,omitempty"`
	StatusMessage *string `json:"statusMessage,omitempty"`
}

type AzureSubscription struct {
	DisplayName            *string `json:"displayName,omitempty"`
	SubscriptionId         *string `json:"subscriptionId,omitempty"`
	SubscriptionTenantId   *string `json:"subscriptionTenantId,omitempty"`
	SubscriptionTenantName *string `json:"subscriptionTenantName,omitempty"`
}

type AzureSubscriptionQueryResult struct {
	ErrorMessage *string              `json:"errorMessage,omitempty"`
	Value        *[]AzureSubscription `json:"value,omitempty"`
}

type ClientCertificate struct {
	// Gets or sets the value of client certificate.
	Value *string `json:"value,omitempty"`
}

type CounterVariable struct {
	Prefix *string `json:"prefix,omitempty"`
	Seed   *int    `json:"seed,omitempty"`
	Value  *int    `json:"value,omitempty"`
}

type DataSource struct {
	AuthenticationScheme *AuthenticationSchemeReference `json:"authenticationScheme,omitempty"`
	EndpointUrl          *string                        `json:"endpointUrl,omitempty"`
	Headers              *[]AuthorizationHeader         `json:"headers,omitempty"`
	Name                 *string                        `json:"name,omitempty"`
	ResourceUrl          *string                        `json:"resourceUrl,omitempty"`
	ResultSelector       *string                        `json:"resultSelector,omitempty"`
}

type DataSourceBinding struct {
	// Pagination format supported by this data source(ContinuationToken/SkipTop).
	CallbackContextTemplate *string `json:"callbackContextTemplate,omitempty"`
	// Subsequent calls needed?
	CallbackRequiredTemplate *string `json:"callbackRequiredTemplate,omitempty"`
	// Gets or sets the name of the data source.
	DataSourceName *string `json:"dataSourceName,omitempty"`
	// Gets or sets the endpoint Id.
	EndpointId *string `json:"endpointId,omitempty"`
	// Gets or sets the url of the service endpoint.
	EndpointUrl *string `json:"endpointUrl,omitempty"`
	// Gets or sets the authorization headers.
	Headers *[]distributedtaskcommon.AuthorizationHeader `json:"headers,omitempty"`
	// Defines the initial value of the query params
	InitialContextTemplate *string `json:"initialContextTemplate,omitempty"`
	// Gets or sets the parameters for the data source.
	Parameters *map[string]string `json:"parameters,omitempty"`
	// Gets or sets http request body
	RequestContent *string `json:"requestContent,omitempty"`
	// Gets or sets http request verb
	RequestVerb *string `json:"requestVerb,omitempty"`
	// Gets or sets the result selector.
	ResultSelector *string `json:"resultSelector,omitempty"`
	// Gets or sets the result template.
	ResultTemplate *string `json:"resultTemplate,omitempty"`
	// Gets or sets the target of the data source.
	Target *string `json:"target,omitempty"`
}

type DataSourceDetails struct {
	DataSourceName *string                `json:"dataSourceName,omitempty"`
	DataSourceUrl  *string                `json:"dataSourceUrl,omitempty"`
	Headers        *[]AuthorizationHeader `json:"headers,omitempty"`
	Parameters     *map[string]string     `json:"parameters,omitempty"`
	ResourceUrl    *string                `json:"resourceUrl,omitempty"`
	ResultSelector *string                `json:"resultSelector,omitempty"`
}

type Demand struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

type DemandEquals struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

type DemandExists struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

type DemandMinimumVersion struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

type DependencyBinding struct {
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
}

type DependencyData struct {
	Input *string                     `json:"input,omitempty"`
	Map   *[]azuredevops.KeyValuePair `json:"map,omitempty"`
}

type DependsOn struct {
	Input *string              `json:"input,omitempty"`
	Map   *[]DependencyBinding `json:"map,omitempty"`
}

type DeploymentGatesChangeEvent struct {
	GateNames *[]string `json:"gateNames,omitempty"`
}

// Deployment group.
type DeploymentGroup struct {
	// Deployment group identifier.
	Id *int `json:"id,omitempty"`
	// Name of the deployment group.
	Name *string `json:"name,omitempty"`
	// Deployment pool in which deployment agents are registered.
	Pool *TaskAgentPoolReference `json:"pool,omitempty"`
	// Project to which the deployment group belongs.
	Project *ProjectReference `json:"project,omitempty"`
	// Description of the deployment group.
	Description *string `json:"description,omitempty"`
	// Number of deployment targets in the deployment group.
	MachineCount *int `json:"machineCount,omitempty"`
	// List of deployment targets in the deployment group.
	Machines *[]DeploymentMachine `json:"machines,omitempty"`
	// List of unique tags across all deployment targets in the deployment group.
	MachineTags *[]string `json:"machineTags,omitempty"`
}

// [Flags] This is useful in getting a list of deployment groups, filtered for which caller has permissions to take a particular action.
type DeploymentGroupActionFilter string

type deploymentGroupActionFilterValuesType struct {
	None   DeploymentGroupActionFilter
	Manage DeploymentGroupActionFilter
	Use    DeploymentGroupActionFilter
}

var DeploymentGroupActionFilterValues = deploymentGroupActionFilterValuesType{
	// All deployment groups.
	None: "none",
	// Only deployment groups for which caller has **manage** permission.
	Manage: "manage",
	// Only deployment groups for which caller has **use** permission.
	Use: "use",
}

// Properties to create Deployment group.
type DeploymentGroupCreateParameter struct {
	// Description of the deployment group.
	Description *string `json:"description,omitempty"`
	// Name of the deployment group.
	Name *string `json:"name,omitempty"`
	// Identifier of the deployment pool in which deployment agents are registered.
	PoolId *int `json:"poolId,omitempty"`
}

// Properties of Deployment pool to create Deployment group.
type DeploymentGroupCreateParameterPoolProperty struct {
	// Deployment pool identifier.
	Id *int `json:"id,omitempty"`
}

// [Flags] Properties to be included or expanded in deployment group objects. This is useful when getting a single or list of deployment grouops.
type DeploymentGroupExpands string

type deploymentGroupExpandsValuesType struct {
	None     DeploymentGroupExpands
	Machines DeploymentGroupExpands
	Tags     DeploymentGroupExpands
}

var DeploymentGroupExpandsValues = deploymentGroupExpandsValuesType{
	// No additional properties.
	None: "none",
	// Deprecated: Include all the deployment targets.
	Machines: "machines",
	// Include unique list of tags across all deployment targets.
	Tags: "tags",
}

// Deployment group metrics.
type DeploymentGroupMetrics struct {
	// List of deployment group properties. And types of metrics provided for those properties.
	ColumnsHeader *MetricsColumnsHeader `json:"columnsHeader,omitempty"`
	// Deployment group.
	DeploymentGroup *DeploymentGroupReference `json:"deploymentGroup,omitempty"`
	// Values of properties and the metrics. E.g. 1: total count of deployment targets for which 'TargetState' is 'offline'. E.g. 2: Average time of deployment to the deployment targets for which 'LastJobStatus' is 'passed' and 'TargetState' is 'online'.
	Rows *[]MetricsRow `json:"rows,omitempty"`
}

// Deployment group reference. This is useful for referring a deployment group in another object.
type DeploymentGroupReference struct {
	// Deployment group identifier.
	Id *int `json:"id,omitempty"`
	// Name of the deployment group.
	Name *string `json:"name,omitempty"`
	// Deployment pool in which deployment agents are registered.
	Pool *TaskAgentPoolReference `json:"pool,omitempty"`
	// Project to which the deployment group belongs.
	Project *ProjectReference `json:"project,omitempty"`
}

// Deployment group update parameter.
type DeploymentGroupUpdateParameter struct {
	// Description of the deployment group.
	Description *string `json:"description,omitempty"`
	// Name of the deployment group.
	Name *string `json:"name,omitempty"`
}

// Deployment target.
type DeploymentMachine struct {
	// Deployment agent.
	Agent *TaskAgent `json:"agent,omitempty"`
	// Deployment target Identifier.
	Id *int `json:"id,omitempty"`
	// Properties of the deployment target.
	Properties interface{} `json:"properties,omitempty"`
	// Tags of the deployment target.
	Tags *[]string `json:"tags,omitempty"`
}

type DeploymentMachineChangedData struct {
	// Deployment agent.
	Agent *TaskAgent `json:"agent,omitempty"`
	// Deployment target Identifier.
	Id *int `json:"id,omitempty"`
	// Properties of the deployment target.
	Properties interface{} `json:"properties,omitempty"`
	// Tags of the deployment target.
	Tags        *[]string `json:"tags,omitempty"`
	AddedTags   *[]string `json:"addedTags,omitempty"`
	DeletedTags *[]string `json:"deletedTags,omitempty"`
}

// [Flags]
type DeploymentMachineExpands string

type deploymentMachineExpandsValuesType struct {
	None            DeploymentMachineExpands
	Capabilities    DeploymentMachineExpands
	AssignedRequest DeploymentMachineExpands
}

var DeploymentMachineExpandsValues = deploymentMachineExpandsValuesType{
	None:            "none",
	Capabilities:    "capabilities",
	AssignedRequest: "assignedRequest",
}

type DeploymentMachineGroup struct {
	Id       *int                    `json:"id,omitempty"`
	Name     *string                 `json:"name,omitempty"`
	Pool     *TaskAgentPoolReference `json:"pool,omitempty"`
	Project  *ProjectReference       `json:"project,omitempty"`
	Machines *[]DeploymentMachine    `json:"machines,omitempty"`
	Size     *int                    `json:"size,omitempty"`
}

type DeploymentMachineGroupReference struct {
	Id      *int                    `json:"id,omitempty"`
	Name    *string                 `json:"name,omitempty"`
	Pool    *TaskAgentPoolReference `json:"pool,omitempty"`
	Project *ProjectReference       `json:"project,omitempty"`
}

type DeploymentMachinesChangeEvent struct {
	MachineGroupReference *DeploymentGroupReference       `json:"machineGroupReference,omitempty"`
	Machines              *[]DeploymentMachineChangedData `json:"machines,omitempty"`
}

// Deployment pool summary.
type DeploymentPoolSummary struct {
	// List of deployment groups referring to the deployment pool.
	DeploymentGroups *[]DeploymentGroupReference `json:"deploymentGroups,omitempty"`
	// Number of deployment agents that are offline.
	OfflineAgentsCount *int `json:"offlineAgentsCount,omitempty"`
	// Number of deployment agents that are online.
	OnlineAgentsCount *int `json:"onlineAgentsCount,omitempty"`
	// Deployment pool.
	Pool *TaskAgentPoolReference `json:"pool,omitempty"`
	// Virtual machine Resource referring in pool.
	Resource *EnvironmentResourceReference `json:"resource,omitempty"`
}

// [Flags] Properties to be included or expanded in deployment pool summary objects. This is useful when getting a single or list of deployment pool summaries.
type DeploymentPoolSummaryExpands string

type deploymentPoolSummaryExpandsValuesType struct {
	None             DeploymentPoolSummaryExpands
	DeploymentGroups DeploymentPoolSummaryExpands
	Resource         DeploymentPoolSummaryExpands
}

var DeploymentPoolSummaryExpandsValues = deploymentPoolSummaryExpandsValuesType{
	// No additional properties
	None: "none",
	// Include deployment groups referring to the deployment pool.
	DeploymentGroups: "deploymentGroups",
	// Include Resource referring to the deployment pool.
	Resource: "resource",
}

// [Flags] Properties to be included or expanded in deployment target objects. This is useful when getting a single or list of deployment targets.
type DeploymentTargetExpands string

type deploymentTargetExpandsValuesType struct {
	None                 DeploymentTargetExpands
	Capabilities         DeploymentTargetExpands
	AssignedRequest      DeploymentTargetExpands
	LastCompletedRequest DeploymentTargetExpands
}

var DeploymentTargetExpandsValues = deploymentTargetExpandsValuesType{
	// No additional properties.
	None: "none",
	// Include capabilities of the deployment agent.
	Capabilities: "capabilities",
	// Include the job request assigned to the deployment agent.
	AssignedRequest: "assignedRequest",
	// Include the last completed job request of the deployment agent.
	LastCompletedRequest: "lastCompletedRequest",
}

// Deployment target update parameter.
type DeploymentTargetUpdateParameter struct {
	// Identifier of the deployment target.
	Id   *int      `json:"id,omitempty"`
	Tags *[]string `json:"tags,omitempty"`
}

type DiagnosticLogMetadata struct {
	AgentId     *int    `json:"agentId,omitempty"`
	AgentName   *string `json:"agentName,omitempty"`
	FileName    *string `json:"fileName,omitempty"`
	PhaseName   *string `json:"phaseName,omitempty"`
	PhaseResult *string `json:"phaseResult,omitempty"`
	PoolId      *int    `json:"poolId,omitempty"`
}

type EndpointAuthorization struct {
	// Gets or sets the parameters for the selected authorization scheme.
	Parameters *map[string]string `json:"parameters,omitempty"`
	// Gets or sets the scheme used for service endpoint authentication.
	Scheme *string `json:"scheme,omitempty"`
}

// Represents url of the service endpoint.
type EndpointUrl struct {
	// Gets or sets the dependency bindings.
	DependsOn *DependsOn `json:"dependsOn,omitempty"`
	// Gets or sets the display name of service endpoint url.
	DisplayName *string `json:"displayName,omitempty"`
	// Gets or sets the help text of service endpoint url.
	HelpText *string `json:"helpText,omitempty"`
	// Gets or sets the visibility of service endpoint url.
	IsVisible *string `json:"isVisible,omitempty"`
	// Gets or sets the value of service endpoint url.
	Value *string `json:"value,omitempty"`
}

// [Flags] This is useful in getting a list of Environments, filtered for which caller has permissions to take a particular action.
type EnvironmentActionFilter string

type environmentActionFilterValuesType struct {
	None   EnvironmentActionFilter
	Manage EnvironmentActionFilter
	Use    EnvironmentActionFilter
}

var EnvironmentActionFilterValues = environmentActionFilterValuesType{
	// All environments for which user has **view** permission.
	None: "none",
	// Only environments for which caller has **manage** permission.
	Manage: "manage",
	// Only environments for which caller has **use** permission.
	Use: "use",
}

// Properties to create Environment.
type EnvironmentCreateParameter struct {
	// Description of the environment.
	Description *string `json:"description,omitempty"`
	// Name of the environment.
	Name *string `json:"name,omitempty"`
}

// EnvironmentDeploymentExecutionRecord.
type EnvironmentDeploymentExecutionRecord struct {
	// Definition of the environment deployment execution owner
	Definition *TaskOrchestrationOwner `json:"definition,omitempty"`
	// Id of the Environment
	EnvironmentId *int `json:"environmentId,omitempty"`
	// Finish time of the environment deployment execution
	FinishTime *azuredevops.Time `json:"finishTime,omitempty"`
	// Id of the Environment deployment execution history record
	Id *uint64 `json:"id,omitempty"`
	// Job Attempt
	JobAttempt *int `json:"jobAttempt,omitempty"`
	// Job name
	JobName *string `json:"jobName,omitempty"`
	// Owner of the environment deployment execution record
	Owner *TaskOrchestrationOwner `json:"owner,omitempty"`
	// Plan Id
	PlanId *uuid.UUID `json:"planId,omitempty"`
	// Plan type of the environment deployment execution record
	PlanType *string `json:"planType,omitempty"`
	// Queue time of the environment deployment execution
	QueueTime *azuredevops.Time `json:"queueTime,omitempty"`
	// Request identifier of the Environment deployment execution history record
	RequestIdentifier *string `json:"requestIdentifier,omitempty"`
	// Resource Id
	ResourceId *int `json:"resourceId,omitempty"`
	// Result of the environment deployment execution
	Result *TaskResult `json:"result,omitempty"`
	// Project Id
	ScopeId *uuid.UUID `json:"scopeId,omitempty"`
	// Service owner Id
	ServiceOwner *uuid.UUID `json:"serviceOwner,omitempty"`
	// Stage Attempt
	StageAttempt *int `json:"stageAttempt,omitempty"`
	// Stage name
	StageName *string `json:"stageName,omitempty"`
	// Start time of the environment deployment execution
	StartTime *azuredevops.Time `json:"startTime,omitempty"`
}

// [Flags] Properties to be included or expanded in environment objects. This is useful when getting a single environment.
type EnvironmentExpands string

type environmentExpandsValuesType struct {
	None               EnvironmentExpands
	ResourceReferences EnvironmentExpands
}

var EnvironmentExpandsValues = environmentExpandsValuesType{
	// No additional properties
	None: "none",
	// Include resource references referring to the environment.
	ResourceReferences: "resourceReferences",
}

// Environment.
type EnvironmentInstance struct {
	// Identity reference of the user who created the Environment.
	CreatedBy *webapi.IdentityRef `json:"createdBy,omitempty"`
	// Creation time of the Environment
	CreatedOn *azuredevops.Time `json:"createdOn,omitempty"`
	// Description of the Environment.
	Description *string `json:"description,omitempty"`
	// Id of the Environment
	Id *int `json:"id,omitempty"`
	// Identity reference of the user who last modified the Environment.
	LastModifiedBy *webapi.IdentityRef `json:"lastModifiedBy,omitempty"`
	// Last modified time of the Environment
	LastModifiedOn *azuredevops.Time `json:"lastModifiedOn,omitempty"`
	// Name of the Environment.
	Name      *string                         `json:"name,omitempty"`
	Resources *[]EnvironmentResourceReference `json:"resources,omitempty"`
}

// EnvironmentLinkedResourceReference.
type EnvironmentLinkedResourceReference struct {
	// Id of the resource.
	Id *string `json:"id,omitempty"`
	// Type of resource.
	TypeName *string `json:"typeName,omitempty"`
}

type EnvironmentReference struct {
	Id   *int    `json:"id,omitempty"`
	Name *string `json:"name,omitempty"`
}

type EnvironmentResource struct {
	CreatedBy            *webapi.IdentityRef   `json:"createdBy,omitempty"`
	CreatedOn            *azuredevops.Time     `json:"createdOn,omitempty"`
	EnvironmentReference *EnvironmentReference `json:"environmentReference,omitempty"`
	Id                   *int                  `json:"id,omitempty"`
	LastModifiedBy       *webapi.IdentityRef   `json:"lastModifiedBy,omitempty"`
	LastModifiedOn       *azuredevops.Time     `json:"lastModifiedOn,omitempty"`
	Name                 *string               `json:"name,omitempty"`
	// Environment resource type
	Type *EnvironmentResourceType `json:"type,omitempty"`
}

// EnvironmentResourceReference.
type EnvironmentResourceReference struct {
	// Id of the resource.
	Id *int `json:"id,omitempty"`
	// Name of the resource.
	Name *string `json:"name,omitempty"`
	// Type of the resource.
	Type *EnvironmentResourceType `json:"type,omitempty"`
}

// [Flags] EnvironmentResourceType.
type EnvironmentResourceType string

type environmentResourceTypeValuesType struct {
	Undefined      EnvironmentResourceType
	Generic        EnvironmentResourceType
	VirtualMachine EnvironmentResourceType
	Kubernetes     EnvironmentResourceType
}

var EnvironmentResourceTypeValues = environmentResourceTypeValuesType{
	Undefined: "undefined",
	// Unknown resource type
	Generic: "generic",
	// Virtual machine resource type
	VirtualMachine: "virtualMachine",
	// Kubernetes resource type
	Kubernetes: "kubernetes",
}

// Properties to update Environment.
type EnvironmentUpdateParameter struct {
	// Description of the environment.
	Description *string `json:"description,omitempty"`
	// Name of the environment.
	Name *string `json:"name,omitempty"`
}

type EventsConfig struct {
}

type ExpressionValidationItem struct {
	// Tells whether the current input is valid or not
	IsValid *bool `json:"isValid,omitempty"`
	// Reason for input validation failure
	Reason *string `json:"reason,omitempty"`
	// Type of validation item
	Type *string `json:"type,omitempty"`
	// Value to validate. The conditional expression to validate for the input for "expression" type Eg:eq(variables['Build.SourceBranch'], 'refs/heads/master');eq(value, 'refs/heads/master')
	Value *string `json:"value,omitempty"`
}

type HelpLink struct {
	Text *string `json:"text,omitempty"`
	Url  *string `json:"url,omitempty"`
}

type InputBindingContext struct {
	// Value of the input
	Value *string `json:"value,omitempty"`
}

type InputValidationItem struct {
	// Tells whether the current input is valid or not
	IsValid *bool `json:"isValid,omitempty"`
	// Reason for input validation failure
	Reason *string `json:"reason,omitempty"`
	// Type of validation item
	Type *string `json:"type,omitempty"`
	// Value to validate. The conditional expression to validate for the input for "expression" type Eg:eq(variables['Build.SourceBranch'], 'refs/heads/master');eq(value, 'refs/heads/master')
	Value *string `json:"value,omitempty"`
	// Provides binding context for the expression to evaluate
	Context *InputBindingContext `json:"context,omitempty"`
}

type InputValidationRequest struct {
	Inputs *map[string]ValidationItem `json:"inputs,omitempty"`
}

type Issue struct {
	Category *string            `json:"category,omitempty"`
	Data     *map[string]string `json:"data,omitempty"`
	Message  *string            `json:"message,omitempty"`
	Type     *IssueType         `json:"type,omitempty"`
}

type IssueType string

type issueTypeValuesType struct {
	Error   IssueType
	Warning IssueType
}

var IssueTypeValues = issueTypeValuesType{
	Error:   "error",
	Warning: "warning",
}

type JobAssignedEvent struct {
	JobId   *uuid.UUID           `json:"jobId,omitempty"`
	Name    *string              `json:"name,omitempty"`
	Request *TaskAgentJobRequest `json:"request,omitempty"`
}

type JobCancelMessage struct {
	JobId   *uuid.UUID  `json:"jobId,omitempty"`
	Timeout interface{} `json:"timeout,omitempty"`
}

type JobCompletedEvent struct {
	JobId     *uuid.UUID  `json:"jobId,omitempty"`
	Name      *string     `json:"name,omitempty"`
	RequestId *uint64     `json:"requestId,omitempty"`
	Result    *TaskResult `json:"result,omitempty"`
}

// Represents the context of variables and vectors for a job request.
type JobEnvironment struct {
	Endpoints   *[]ServiceEndpoint       `json:"endpoints,omitempty"`
	Mask        *[]MaskHint              `json:"mask,omitempty"`
	Options     *map[uuid.UUID]JobOption `json:"options,omitempty"`
	SecureFiles *[]SecureFile            `json:"secureFiles,omitempty"`
	// Gets or sets the endpoint used for communicating back to the calling service.
	SystemConnection *ServiceEndpoint   `json:"systemConnection,omitempty"`
	Variables        *map[string]string `json:"variables,omitempty"`
}

type JobEvent struct {
	JobId *uuid.UUID `json:"jobId,omitempty"`
	Name  *string    `json:"name,omitempty"`
}

type JobEventConfig struct {
	Timeout *string `json:"timeout,omitempty"`
}

type JobEventsConfig struct {
	JobAssigned  *JobEventConfig `json:"jobAssigned,omitempty"`
	JobCompleted *JobEventConfig `json:"jobCompleted,omitempty"`
	JobStarted   *JobEventConfig `json:"jobStarted,omitempty"`
}

// Represents an option that may affect the way an agent runs the job.
type JobOption struct {
	Data *map[string]string `json:"data,omitempty"`
	// Gets the id of the option.
	Id *uuid.UUID `json:"id,omitempty"`
}

type JobRequestMessage struct {
	Environment *JobEnvironment                 `json:"environment,omitempty"`
	JobId       *uuid.UUID                      `json:"jobId,omitempty"`
	JobName     *string                         `json:"jobName,omitempty"`
	JobRefName  *string                         `json:"jobRefName,omitempty"`
	MessageType *string                         `json:"messageType,omitempty"`
	Plan        *TaskOrchestrationPlanReference `json:"plan,omitempty"`
	Timeline    *TimelineReference              `json:"timeline,omitempty"`
}

type JobStartedEvent struct {
	JobId *uuid.UUID `json:"jobId,omitempty"`
	Name  *string    `json:"name,omitempty"`
}

type KubernetesResource struct {
	CreatedBy            *webapi.IdentityRef   `json:"createdBy,omitempty"`
	CreatedOn            *azuredevops.Time     `json:"createdOn,omitempty"`
	EnvironmentReference *EnvironmentReference `json:"environmentReference,omitempty"`
	Id                   *int                  `json:"id,omitempty"`
	LastModifiedBy       *webapi.IdentityRef   `json:"lastModifiedBy,omitempty"`
	LastModifiedOn       *azuredevops.Time     `json:"lastModifiedOn,omitempty"`
	Name                 *string               `json:"name,omitempty"`
	// Environment resource type
	Type              *EnvironmentResourceType `json:"type,omitempty"`
	ClusterName       *string                  `json:"clusterName,omitempty"`
	Namespace         *string                  `json:"namespace,omitempty"`
	ServiceEndpointId *uuid.UUID               `json:"serviceEndpointId,omitempty"`
}

type KubernetesResourceCreateParameters struct {
	ClusterName       *string    `json:"clusterName,omitempty"`
	Name              *string    `json:"name,omitempty"`
	Namespace         *string    `json:"namespace,omitempty"`
	ServiceEndpointId *uuid.UUID `json:"serviceEndpointId,omitempty"`
}

// [Flags]
type MachineGroupActionFilter string

type machineGroupActionFilterValuesType struct {
	None   MachineGroupActionFilter
	Manage MachineGroupActionFilter
	Use    MachineGroupActionFilter
}

var MachineGroupActionFilterValues = machineGroupActionFilterValuesType{
	None:   "none",
	Manage: "manage",
	Use:    "use",
}

// Represents a purchase of resource units in a secondary marketplace.
type MarketplacePurchasedLicense struct {
	// The Marketplace display name.
	MarketplaceName *string `json:"marketplaceName,omitempty"`
	// The name of the identity making the purchase as seen by the marketplace
	PurchaserName *string `json:"purchaserName,omitempty"`
	// The quantity purchased.
	PurchaseUnitCount *int `json:"purchaseUnitCount,omitempty"`
}

type MaskHint struct {
	Type  *MaskType `json:"type,omitempty"`
	Value *string   `json:"value,omitempty"`
}

type MaskType string

type maskTypeValuesType struct {
	Variable MaskType
	Regex    MaskType
}

var MaskTypeValues = maskTypeValuesType{
	Variable: "variable",
	Regex:    "regex",
}

// Meta data for a metrics column.
type MetricsColumnMetaData struct {
	// Name.
	ColumnName *string `json:"columnName,omitempty"`
	// Data type.
	ColumnValueType *string `json:"columnValueType,omitempty"`
}

// Metrics columns header
type MetricsColumnsHeader struct {
	// Properties of deployment group for which metrics are provided. E.g. 1: LastJobStatus E.g. 2: TargetState
	Dimensions *[]MetricsColumnMetaData `json:"dimensions,omitempty"`
	// The types of metrics. E.g. 1: total count of deployment targets. E.g. 2: Average time of deployment to the deployment targets.
	Metrics *[]MetricsColumnMetaData `json:"metrics,omitempty"`
}

// Metrics row.
type MetricsRow struct {
	// The values of the properties mentioned as 'Dimensions' in column header. E.g. 1: For a property 'LastJobStatus' - metrics will be provided for 'passed', 'failed', etc. E.g. 2: For a property 'TargetState' - metrics will be provided for 'online', 'offline' targets.
	Dimensions *[]string `json:"dimensions,omitempty"`
	// Metrics in serialized format. Should be deserialized based on the data type provided in header.
	Metrics *[]string `json:"metrics,omitempty"`
}

// Represents a downloadable package.
type PackageMetadata struct {
	// The date the package was created
	CreatedOn *azuredevops.Time `json:"createdOn,omitempty"`
	// A direct link to download the package.
	DownloadUrl *string `json:"downloadUrl,omitempty"`
	// The UI uses this to display instructions, i.e. "unzip MyAgent.zip"
	Filename *string `json:"filename,omitempty"`
	// MD5 hash as a base64 string
	HashValue *string `json:"hashValue,omitempty"`
	// A link to documentation
	InfoUrl *string `json:"infoUrl,omitempty"`
	// The platform (win7, linux, etc.)
	Platform *string `json:"platform,omitempty"`
	// The type of package (e.g. "agent")
	Type *string `json:"type,omitempty"`
	// The package version.
	Version *PackageVersion `json:"version,omitempty"`
}

type PackageVersion struct {
	Major *int `json:"major,omitempty"`
	Minor *int `json:"minor,omitempty"`
	Patch *int `json:"patch,omitempty"`
}

type PlanEnvironment struct {
	Mask      *[]MaskHint              `json:"mask,omitempty"`
	Options   *map[uuid.UUID]JobOption `json:"options,omitempty"`
	Variables *map[string]string       `json:"variables,omitempty"`
}

// [Flags]
type PlanGroupStatus string

type planGroupStatusValuesType struct {
	Running PlanGroupStatus
	Queued  PlanGroupStatus
	All     PlanGroupStatus
}

var PlanGroupStatusValues = planGroupStatusValuesType{
	Running: "running",
	Queued:  "queued",
	All:     "all",
}

// [Flags]
type PlanGroupStatusFilter string

type planGroupStatusFilterValuesType struct {
	Running PlanGroupStatusFilter
	Queued  PlanGroupStatusFilter
	All     PlanGroupStatusFilter
}

var PlanGroupStatusFilterValues = planGroupStatusFilterValuesType{
	Running: "running",
	Queued:  "queued",
	All:     "all",
}

type ProjectReference struct {
	Id   *uuid.UUID `json:"id,omitempty"`
	Name *string    `json:"name,omitempty"`
}

type PublishTaskGroupMetadata struct {
	Comment                  *string    `json:"comment,omitempty"`
	ParentDefinitionRevision *int       `json:"parentDefinitionRevision,omitempty"`
	Preview                  *bool      `json:"preview,omitempty"`
	TaskGroupId              *uuid.UUID `json:"taskGroupId,omitempty"`
	TaskGroupRevision        *int       `json:"taskGroupRevision,omitempty"`
}

type ResourceFilterOptions struct {
	Identities    *[]webapi.IdentityRef `json:"identities,omitempty"`
	ResourceTypes *[]string             `json:"resourceTypes,omitempty"`
}

type ResourceFilters struct {
	CreatedBy    *[]uuid.UUID `json:"createdBy,omitempty"`
	ResourceType *[]string    `json:"resourceType,omitempty"`
	SearchText   *string      `json:"searchText,omitempty"`
}

// Resources include Service Connections, Variable Groups and Secure Files.
type ResourceItem struct {
	// Gets or sets the identity who created the resource.
	CreatedBy *webapi.IdentityRef `json:"createdBy,omitempty"`
	// Gets or sets description of the resource.
	Description *string `json:"description,omitempty"`
	// Gets or sets icon url of the resource.
	IconUrl *string `json:"iconUrl,omitempty"`
	// Gets or sets Id of the resource.
	Id *string `json:"id,omitempty"`
	// Indicates whether resource is shared with other projects or not.
	IsShared *bool `json:"isShared,omitempty"`
	// Gets or sets name of the resource.
	Name *string `json:"name,omitempty"`
	// Gets or sets internal properties of the resource.
	Properties *map[string]string `json:"properties,omitempty"`
	// Gets or sets resource type.
	ResourceType *string `json:"resourceType,omitempty"`
}

type ResourceLimit struct {
	FailedToReachAllProviders *bool              `json:"failedToReachAllProviders,omitempty"`
	HostId                    *uuid.UUID         `json:"hostId,omitempty"`
	IsHosted                  *bool              `json:"isHosted,omitempty"`
	IsPremium                 *bool              `json:"isPremium,omitempty"`
	ParallelismTag            *string            `json:"parallelismTag,omitempty"`
	ResourceLimitsData        *map[string]string `json:"resourceLimitsData,omitempty"`
	TotalCount                *int               `json:"totalCount,omitempty"`
	TotalMinutes              *int               `json:"totalMinutes,omitempty"`
}

type ResourcesHubData struct {
	ContinuationToken     *string                `json:"continuationToken,omitempty"`
	ResourceFilterOptions *ResourceFilterOptions `json:"resourceFilterOptions,omitempty"`
	ResourceFilters       *ResourceFilters       `json:"resourceFilters,omitempty"`
	ResourceItems         *[]ResourceItem        `json:"resourceItems,omitempty"`
}

type ResourceUsage struct {
	ResourceLimit   *ResourceLimit         `json:"resourceLimit,omitempty"`
	RunningRequests *[]TaskAgentJobRequest `json:"runningRequests,omitempty"`
	UsedCount       *int                   `json:"usedCount,omitempty"`
	UsedMinutes     *int                   `json:"usedMinutes,omitempty"`
}

type ResultTransformationDetails struct {
	ResultTemplate *string `json:"resultTemplate,omitempty"`
}

type SecureFile struct {
	CreatedBy  *webapi.IdentityRef `json:"createdBy,omitempty"`
	CreatedOn  *azuredevops.Time   `json:"createdOn,omitempty"`
	Id         *uuid.UUID          `json:"id,omitempty"`
	ModifiedBy *webapi.IdentityRef `json:"modifiedBy,omitempty"`
	ModifiedOn *azuredevops.Time   `json:"modifiedOn,omitempty"`
	Name       *string             `json:"name,omitempty"`
	Properties *map[string]string  `json:"properties,omitempty"`
	Ticket     *string             `json:"ticket,omitempty"`
}

// [Flags]
type SecureFileActionFilter string

type secureFileActionFilterValuesType struct {
	None   SecureFileActionFilter
	Manage SecureFileActionFilter
	Use    SecureFileActionFilter
}

var SecureFileActionFilterValues = secureFileActionFilterValuesType{
	None:   "none",
	Manage: "manage",
	Use:    "use",
}

type SecureFileEvent struct {
	EventType   *string       `json:"eventType,omitempty"`
	ProjectId   *uuid.UUID    `json:"projectId,omitempty"`
	SecureFiles *[]SecureFile `json:"secureFiles,omitempty"`
}

type SendJobResponse struct {
	Events    *JobEventsConfig   `json:"events,omitempty"`
	Variables *map[string]string `json:"variables,omitempty"`
}

type ServerExecutionDefinition struct {
	Events      *EventsConfig `json:"events,omitempty"`
	HandlerName *string       `json:"handlerName,omitempty"`
}

type ServerTaskRequestMessage struct {
	Environment    *JobEnvironment                 `json:"environment,omitempty"`
	JobId          *uuid.UUID                      `json:"jobId,omitempty"`
	JobName        *string                         `json:"jobName,omitempty"`
	JobRefName     *string                         `json:"jobRefName,omitempty"`
	MessageType    *string                         `json:"messageType,omitempty"`
	Plan           *TaskOrchestrationPlanReference `json:"plan,omitempty"`
	Timeline       *TimelineReference              `json:"timeline,omitempty"`
	TaskDefinition *TaskDefinition                 `json:"taskDefinition,omitempty"`
	TaskInstance   *TaskInstance                   `json:"taskInstance,omitempty"`
}

// Represents an endpoint which may be used by an orchestration job.
type ServiceEndpoint struct {
	// Gets or sets the identity reference for the administrators group of the service endpoint.
	AdministratorsGroup *webapi.IdentityRef `json:"administratorsGroup,omitempty"`
	// Gets or sets the authorization data for talking to the endpoint.
	Authorization *EndpointAuthorization `json:"authorization,omitempty"`
	// Gets or sets the identity reference for the user who created the Service endpoint.
	CreatedBy *webapi.IdentityRef `json:"createdBy,omitempty"`
	Data      *map[string]string  `json:"data,omitempty"`
	// Gets or sets the description of endpoint.
	Description  *string    `json:"description,omitempty"`
	GroupScopeId *uuid.UUID `json:"groupScopeId,omitempty"`
	// Gets or sets the identifier of this endpoint.
	Id *uuid.UUID `json:"id,omitempty"`
	// EndPoint state indicator
	IsReady *bool `json:"isReady,omitempty"`
	// Indicates whether service endpoint is shared with other projects or not.
	IsShared *bool `json:"isShared,omitempty"`
	// Gets or sets the friendly name of the endpoint.
	Name *string `json:"name,omitempty"`
	// Error message during creation/deletion of endpoint
	OperationStatus interface{} `json:"operationStatus,omitempty"`
	// Gets or sets the owner of the endpoint.
	Owner *string `json:"owner,omitempty"`
	// Gets or sets the identity reference for the readers group of the service endpoint.
	ReadersGroup *webapi.IdentityRef `json:"readersGroup,omitempty"`
	// Gets or sets the type of the endpoint.
	Type *string `json:"type,omitempty"`
	// Gets or sets the url of the endpoint.
	Url *string `json:"url,omitempty"`
}

type ServiceEndpointAuthenticationScheme struct {
	// Gets or sets the authorization headers of service endpoint authentication scheme.
	AuthorizationHeaders *[]AuthorizationHeader `json:"authorizationHeaders,omitempty"`
	// Gets or sets the certificates of service endpoint authentication scheme.
	ClientCertificates *[]ClientCertificate `json:"clientCertificates,omitempty"`
	// Gets or sets the display name for the service endpoint authentication scheme.
	DisplayName *string `json:"displayName,omitempty"`
	// Gets or sets the input descriptors for the service endpoint authentication scheme.
	InputDescriptors *[]forminput.InputDescriptor `json:"inputDescriptors,omitempty"`
	// Gets or sets the scheme for service endpoint authentication.
	Scheme *string `json:"scheme,omitempty"`
}

type ServiceEndpointDetails struct {
	Authorization *EndpointAuthorization `json:"authorization,omitempty"`
	Data          *map[string]string     `json:"data,omitempty"`
	Type          *string                `json:"type,omitempty"`
	Url           *string                `json:"url,omitempty"`
}

// Represents service endpoint execution data.
type ServiceEndpointExecutionData struct {
	// Gets the definition of service endpoint execution owner.
	Definition *TaskOrchestrationOwner `json:"definition,omitempty"`
	// Gets the finish time of service endpoint execution.
	FinishTime *azuredevops.Time `json:"finishTime,omitempty"`
	// Gets the Id of service endpoint execution data.
	Id *uint64 `json:"id,omitempty"`
	// Gets the owner of service endpoint execution data.
	Owner *TaskOrchestrationOwner `json:"owner,omitempty"`
	// Gets the plan type of service endpoint execution data.
	PlanType *string `json:"planType,omitempty"`
	// Gets the result of service endpoint execution.
	Result *TaskResult `json:"result,omitempty"`
	// Gets the start time of service endpoint execution.
	StartTime *azuredevops.Time `json:"startTime,omitempty"`
}

type ServiceEndpointExecutionRecord struct {
	// Gets the execution data of service endpoint execution.
	Data *ServiceEndpointExecutionData `json:"data,omitempty"`
	// Gets the Id of service endpoint.
	EndpointId *uuid.UUID `json:"endpointId,omitempty"`
}

type ServiceEndpointExecutionRecordsInput struct {
	Data        *ServiceEndpointExecutionData `json:"data,omitempty"`
	EndpointIds *[]uuid.UUID                  `json:"endpointIds,omitempty"`
}

type ServiceEndpointRequest struct {
	DataSourceDetails           *DataSourceDetails           `json:"dataSourceDetails,omitempty"`
	ResultTransformationDetails *ResultTransformationDetails `json:"resultTransformationDetails,omitempty"`
	ServiceEndpointDetails      *ServiceEndpointDetails      `json:"serviceEndpointDetails,omitempty"`
}

type ServiceEndpointRequestResult struct {
	ErrorMessage *string     `json:"errorMessage,omitempty"`
	Result       interface{} `json:"result,omitempty"`
	StatusCode   *string     `json:"statusCode,omitempty"`
}

// Represents type of the service endpoint.
type ServiceEndpointType struct {
	// Authentication scheme of service endpoint type.
	AuthenticationSchemes *[]ServiceEndpointAuthenticationScheme `json:"authenticationSchemes,omitempty"`
	// Data sources of service endpoint type.
	DataSources *[]DataSource `json:"dataSources,omitempty"`
	// Dependency data of service endpoint type.
	DependencyData *[]DependencyData `json:"dependencyData,omitempty"`
	// Gets or sets the description of service endpoint type.
	Description *string `json:"description,omitempty"`
	// Gets or sets the display name of service endpoint type.
	DisplayName *string `json:"displayName,omitempty"`
	// Gets or sets the endpoint url of service endpoint type.
	EndpointUrl *EndpointUrl `json:"endpointUrl,omitempty"`
	// Gets or sets the help link of service endpoint type.
	HelpLink     *HelpLink `json:"helpLink,omitempty"`
	HelpMarkDown *string   `json:"helpMarkDown,omitempty"`
	// Gets or sets the icon url of service endpoint type.
	IconUrl *string `json:"iconUrl,omitempty"`
	// Input descriptor of service endpoint type.
	InputDescriptors *[]forminput.InputDescriptor `json:"inputDescriptors,omitempty"`
	// Gets or sets the name of service endpoint type.
	Name *string `json:"name,omitempty"`
	// Trusted hosts of a service endpoint type.
	TrustedHosts *[]string `json:"trustedHosts,omitempty"`
	// Gets or sets the ui contribution id of service endpoint type.
	UiContributionId *string `json:"uiContributionId,omitempty"`
}

// A task agent.
type TaskAgent struct {
	Links interface{} `json:"_links,omitempty"`
	// This agent's access point.
	AccessPoint *string `json:"accessPoint,omitempty"`
	// Whether or not this agent should run jobs.
	Enabled *bool `json:"enabled,omitempty"`
	// Identifier of the agent.
	Id *int `json:"id,omitempty"`
	// Name of the agent.
	Name *string `json:"name,omitempty"`
	// Agent OS.
	OsDescription *string `json:"osDescription,omitempty"`
	// Provisioning state of this agent.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// Whether or not the agent is online.
	Status *TaskAgentStatus `json:"status,omitempty"`
	// Agent version.
	Version *string `json:"version,omitempty"`
	// The agent cloud request that's currently associated with this agent.
	AssignedAgentCloudRequest *TaskAgentCloudRequest `json:"assignedAgentCloudRequest,omitempty"`
	// The request which is currently assigned to this agent.
	AssignedRequest *TaskAgentJobRequest `json:"assignedRequest,omitempty"`
	// Authorization information for this agent.
	Authorization *TaskAgentAuthorization `json:"authorization,omitempty"`
	// Date on which this agent was created.
	CreatedOn *azuredevops.Time `json:"createdOn,omitempty"`
	// The last request which was completed by this agent.
	LastCompletedRequest *TaskAgentJobRequest `json:"lastCompletedRequest,omitempty"`
	// Maximum job parallelism allowed for this agent.
	MaxParallelism *int `json:"maxParallelism,omitempty"`
	// Pending update for this agent.
	PendingUpdate *TaskAgentUpdate `json:"pendingUpdate,omitempty"`
	Properties    interface{}      `json:"properties,omitempty"`
	// Date on which the last connectivity status change occurred.
	StatusChangedOn    *azuredevops.Time  `json:"statusChangedOn,omitempty"`
	SystemCapabilities *map[string]string `json:"systemCapabilities,omitempty"`
	UserCapabilities   *map[string]string `json:"userCapabilities,omitempty"`
}

// Provides data necessary for authorizing the agent using OAuth 2.0 authentication flows.
type TaskAgentAuthorization struct {
	// Endpoint used to obtain access tokens from the configured token service.
	AuthorizationUrl *string `json:"authorizationUrl,omitempty"`
	// Client identifier for this agent.
	ClientId *uuid.UUID `json:"clientId,omitempty"`
	// Public key used to verify the identity of this agent.
	PublicKey *TaskAgentPublicKey `json:"publicKey,omitempty"`
}

type TaskAgentCloud struct {
	// Gets or sets a AcquireAgentEndpoint using which a request can be made to acquire new agent
	AcquireAgentEndpoint          *string    `json:"acquireAgentEndpoint,omitempty"`
	AcquisitionTimeout            *int       `json:"acquisitionTimeout,omitempty"`
	AgentCloudId                  *int       `json:"agentCloudId,omitempty"`
	GetAccountParallelismEndpoint *string    `json:"getAccountParallelismEndpoint,omitempty"`
	GetAgentDefinitionEndpoint    *string    `json:"getAgentDefinitionEndpoint,omitempty"`
	GetAgentRequestStatusEndpoint *string    `json:"getAgentRequestStatusEndpoint,omitempty"`
	Id                            *uuid.UUID `json:"id,omitempty"`
	// Signifies that this Agent Cloud is internal and should not be user-manageable
	Internal             *bool   `json:"internal,omitempty"`
	MaxParallelism       *int    `json:"maxParallelism,omitempty"`
	Name                 *string `json:"name,omitempty"`
	ReleaseAgentEndpoint *string `json:"releaseAgentEndpoint,omitempty"`
	SharedSecret         *string `json:"sharedSecret,omitempty"`
	// Gets or sets the type of the endpoint.
	Type *string `json:"type,omitempty"`
}

type TaskAgentCloudRequest struct {
	Agent                *TaskAgentReference     `json:"agent,omitempty"`
	AgentCloudId         *int                    `json:"agentCloudId,omitempty"`
	AgentConnectedTime   *azuredevops.Time       `json:"agentConnectedTime,omitempty"`
	AgentData            interface{}             `json:"agentData,omitempty"`
	AgentSpecification   interface{}             `json:"agentSpecification,omitempty"`
	Pool                 *TaskAgentPoolReference `json:"pool,omitempty"`
	ProvisionedTime      *azuredevops.Time       `json:"provisionedTime,omitempty"`
	ProvisionRequestTime *azuredevops.Time       `json:"provisionRequestTime,omitempty"`
	ReleaseRequestTime   *azuredevops.Time       `json:"releaseRequestTime,omitempty"`
	RequestId            *uuid.UUID              `json:"requestId,omitempty"`
}

type TaskAgentCloudType struct {
	// Gets or sets the display name of agent cloud type.
	DisplayName *string `json:"displayName,omitempty"`
	// Gets or sets the input descriptors
	InputDescriptors *[]forminput.InputDescriptor `json:"inputDescriptors,omitempty"`
	// Gets or sets the name of agent cloud type.
	Name *string `json:"name,omitempty"`
}

type TaskAgentDelaySource struct {
	Delays    *[]interface{}      `json:"delays,omitempty"`
	TaskAgent *TaskAgentReference `json:"taskAgent,omitempty"`
}

type TaskAgentJob struct {
	Container         *string                 `json:"container,omitempty"`
	Id                *uuid.UUID              `json:"id,omitempty"`
	Name              *string                 `json:"name,omitempty"`
	SidecarContainers *map[string]string      `json:"sidecarContainers,omitempty"`
	Steps             *[]TaskAgentJobStep     `json:"steps,omitempty"`
	Variables         *[]TaskAgentJobVariable `json:"variables,omitempty"`
}

// A job request for an agent.
type TaskAgentJobRequest struct {
	AgentDelays        *[]TaskAgentDelaySource `json:"agentDelays,omitempty"`
	AgentSpecification interface{}             `json:"agentSpecification,omitempty"`
	// The date/time this request was assigned.
	AssignTime *azuredevops.Time `json:"assignTime,omitempty"`
	// Additional data about the request.
	Data *map[string]string `json:"data,omitempty"`
	// The pipeline definition associated with this request
	Definition *TaskOrchestrationOwner `json:"definition,omitempty"`
	// A list of demands required to fulfill this request.
	Demands          *[]interface{} `json:"demands,omitempty"`
	ExpectedDuration interface{}    `json:"expectedDuration,omitempty"`
	// The date/time this request was finished.
	FinishTime *azuredevops.Time `json:"finishTime,omitempty"`
	// The host which triggered this request.
	HostId *uuid.UUID `json:"hostId,omitempty"`
	// ID of the job resulting from this request.
	JobId *uuid.UUID `json:"jobId,omitempty"`
	// Name of the job resulting from this request.
	JobName *string `json:"jobName,omitempty"`
	// The deadline for the agent to renew the lock.
	LockedUntil            *azuredevops.Time     `json:"lockedUntil,omitempty"`
	MatchedAgents          *[]TaskAgentReference `json:"matchedAgents,omitempty"`
	MatchesAllAgentsInPool *bool                 `json:"matchesAllAgentsInPool,omitempty"`
	OrchestrationId        *string               `json:"orchestrationId,omitempty"`
	// The pipeline associated with this request
	Owner     *TaskOrchestrationOwner `json:"owner,omitempty"`
	PlanGroup *string                 `json:"planGroup,omitempty"`
	// Internal ID for the orchestration plan connected with this request.
	PlanId *uuid.UUID `json:"planId,omitempty"`
	// Internal detail representing the type of orchestration plan.
	PlanType *string `json:"planType,omitempty"`
	// The ID of the pool this request targets
	PoolId *int `json:"poolId,omitempty"`
	// The ID of the queue this request targets
	QueueId *int `json:"queueId,omitempty"`
	// The date/time this request was queued.
	QueueTime *azuredevops.Time `json:"queueTime,omitempty"`
	// The date/time this request was receieved by an agent.
	ReceiveTime *azuredevops.Time `json:"receiveTime,omitempty"`
	// ID of the request.
	RequestId *uint64 `json:"requestId,omitempty"`
	// The agent allocated for this request.
	ReservedAgent *TaskAgentReference `json:"reservedAgent,omitempty"`
	// The result of this request.
	Result *TaskResult `json:"result,omitempty"`
	// Scope of the pipeline; matches the project ID.
	ScopeId *uuid.UUID `json:"scopeId,omitempty"`
	// The service which owns this request.
	ServiceOwner  *uuid.UUID `json:"serviceOwner,omitempty"`
	StatusMessage *string    `json:"statusMessage,omitempty"`
	UserDelayed   *bool      `json:"userDelayed,omitempty"`
}

// [Flags] This is useful in getting a list of deployment targets, filtered by the result of their last job.
type TaskAgentJobResultFilter string

type taskAgentJobResultFilterValuesType struct {
	Failed        TaskAgentJobResultFilter
	Passed        TaskAgentJobResultFilter
	NeverDeployed TaskAgentJobResultFilter
	All           TaskAgentJobResultFilter
}

var TaskAgentJobResultFilterValues = taskAgentJobResultFilterValuesType{
	// Only those deployment targets on which last job failed (**Abandoned**, **Canceled**, **Failed**, **Skipped**).
	Failed: "failed",
	// Only those deployment targets on which last job Passed (**Succeeded**, **Succeeded with issues**).
	Passed: "passed",
	// Only those deployment targets that never executed a job.
	NeverDeployed: "neverDeployed",
	// All deployment targets.
	All: "all",
}

type TaskAgentJobStep struct {
	Condition        *string               `json:"condition,omitempty"`
	ContinueOnError  *bool                 `json:"continueOnError,omitempty"`
	Enabled          *bool                 `json:"enabled,omitempty"`
	Env              *map[string]string    `json:"env,omitempty"`
	Id               *uuid.UUID            `json:"id,omitempty"`
	Inputs           *map[string]string    `json:"inputs,omitempty"`
	Name             *string               `json:"name,omitempty"`
	Task             *TaskAgentJobTask     `json:"task,omitempty"`
	TimeoutInMinutes *int                  `json:"timeoutInMinutes,omitempty"`
	Type             *TaskAgentJobStepType `json:"type,omitempty"`
}

type TaskAgentJobStepType string

type taskAgentJobStepTypeValuesType struct {
	Task   TaskAgentJobStepType
	Action TaskAgentJobStepType
}

var TaskAgentJobStepTypeValues = taskAgentJobStepTypeValuesType{
	Task:   "task",
	Action: "action",
}

type TaskAgentJobTask struct {
	Id      *uuid.UUID `json:"id,omitempty"`
	Name    *string    `json:"name,omitempty"`
	Version *string    `json:"version,omitempty"`
}

type TaskAgentJobVariable struct {
	Name   *string `json:"name,omitempty"`
	Secret *bool   `json:"secret,omitempty"`
	Value  *string `json:"value,omitempty"`
}

type TaskAgentManualUpdate struct {
	Code *TaskAgentUpdateReasonType `json:"code,omitempty"`
}

// Provides a contract for receiving messages from the task orchestrator.
type TaskAgentMessage struct {
	// Gets or sets the body of the message. If the <c>IV</c> property is provided the body will need to be decrypted using the <c>TaskAgentSession.EncryptionKey</c> value in addition to the <c>IV</c>.
	Body *string `json:"body,omitempty"`
	// Gets or sets the initialization vector used to encrypt this message.
	Iv *[]byte `json:"iv,omitempty"`
	// Gets or sets the message identifier.
	MessageId *uint64 `json:"messageId,omitempty"`
	// Gets or sets the message type, describing the data contract found in <c>TaskAgentMessage.Body</c>.
	MessageType *string `json:"messageType,omitempty"`
}

type TaskAgentMinAgentVersionRequiredUpdate struct {
	Code            *TaskAgentUpdateReasonType `json:"code,omitempty"`
	JobDefinition   *TaskOrchestrationOwner    `json:"jobDefinition,omitempty"`
	JobOwner        *TaskOrchestrationOwner    `json:"jobOwner,omitempty"`
	MinAgentVersion interface{}                `json:"minAgentVersion,omitempty"`
}

// An organization-level grouping of agents.
type TaskAgentPool struct {
	Id *int `json:"id,omitempty"`
	// Gets or sets a value indicating whether or not this pool is managed by the service.
	IsHosted *bool `json:"isHosted,omitempty"`
	// Determines whether the pool is legacy.
	IsLegacy *bool   `json:"isLegacy,omitempty"`
	Name     *string `json:"name,omitempty"`
	// Gets or sets the type of the pool
	PoolType *TaskAgentPoolType `json:"poolType,omitempty"`
	Scope    *uuid.UUID         `json:"scope,omitempty"`
	// Gets the current size of the pool.
	Size *int `json:"size,omitempty"`
	// The ID of the associated agent cloud.
	AgentCloudId *int `json:"agentCloudId,omitempty"`
	// Whether or not a queue should be automatically provisioned for each project collection.
	AutoProvision *bool `json:"autoProvision,omitempty"`
	// Whether or not the pool should autosize itself based on the Agent Cloud Provider settings.
	AutoSize *bool `json:"autoSize,omitempty"`
	// Creator of the pool. The creator of the pool is automatically added into the administrators group for the pool on creation.
	CreatedBy *webapi.IdentityRef `json:"createdBy,omitempty"`
	// The date/time of the pool creation.
	CreatedOn *azuredevops.Time `json:"createdOn,omitempty"`
	// Owner or administrator of the pool.
	Owner      *webapi.IdentityRef `json:"owner,omitempty"`
	Properties interface{}         `json:"properties,omitempty"`
	// Target parallelism.
	TargetSize *int `json:"targetSize,omitempty"`
}

// [Flags] Filters pools based on whether the calling user has permission to use or manage the pool.
type TaskAgentPoolActionFilter string

type taskAgentPoolActionFilterValuesType struct {
	None   TaskAgentPoolActionFilter
	Manage TaskAgentPoolActionFilter
	Use    TaskAgentPoolActionFilter
}

var TaskAgentPoolActionFilterValues = taskAgentPoolActionFilterValuesType{
	None:   "none",
	Manage: "manage",
	Use:    "use",
}

type TaskAgentPoolMaintenanceDefinition struct {
	// Enable maintenance
	Enabled *bool `json:"enabled,omitempty"`
	// Id
	Id *int `json:"id,omitempty"`
	// Maintenance job timeout per agent
	JobTimeoutInMinutes *int `json:"jobTimeoutInMinutes,omitempty"`
	// Max percentage of agents within a pool running maintenance job at given time
	MaxConcurrentAgentsPercentage *int                             `json:"maxConcurrentAgentsPercentage,omitempty"`
	Options                       *TaskAgentPoolMaintenanceOptions `json:"options,omitempty"`
	// Pool reference for the maintenance definition
	Pool            *TaskAgentPoolReference                  `json:"pool,omitempty"`
	RetentionPolicy *TaskAgentPoolMaintenanceRetentionPolicy `json:"retentionPolicy,omitempty"`
	ScheduleSetting *TaskAgentPoolMaintenanceSchedule        `json:"scheduleSetting,omitempty"`
}

type TaskAgentPoolMaintenanceJob struct {
	// The maintenance definition for the maintenance job
	DefinitionId *int `json:"definitionId,omitempty"`
	// The total error counts during the maintenance job
	ErrorCount *int `json:"errorCount,omitempty"`
	// Time that the maintenance job was completed
	FinishTime *azuredevops.Time `json:"finishTime,omitempty"`
	// Id of the maintenance job
	JobId *int `json:"jobId,omitempty"`
	// The log download url for the maintenance job
	LogsDownloadUrl *string `json:"logsDownloadUrl,omitempty"`
	// Orchestration/Plan Id for the maintenance job
	OrchestrationId *uuid.UUID `json:"orchestrationId,omitempty"`
	// Pool reference for the maintenance job
	Pool *TaskAgentPoolReference `json:"pool,omitempty"`
	// Time that the maintenance job was queued
	QueueTime *azuredevops.Time `json:"queueTime,omitempty"`
	// The identity that queued the maintenance job
	RequestedBy *webapi.IdentityRef `json:"requestedBy,omitempty"`
	// The maintenance job result
	Result *TaskAgentPoolMaintenanceJobResult `json:"result,omitempty"`
	// Time that the maintenance job was started
	StartTime *azuredevops.Time `json:"startTime,omitempty"`
	// Status of the maintenance job
	Status       *TaskAgentPoolMaintenanceJobStatus        `json:"status,omitempty"`
	TargetAgents *[]TaskAgentPoolMaintenanceJobTargetAgent `json:"targetAgents,omitempty"`
	// The total warning counts during the maintenance job
	WarningCount *int `json:"warningCount,omitempty"`
}

type TaskAgentPoolMaintenanceJobResult string

type taskAgentPoolMaintenanceJobResultValuesType struct {
	Succeeded TaskAgentPoolMaintenanceJobResult
	Failed    TaskAgentPoolMaintenanceJobResult
	Canceled  TaskAgentPoolMaintenanceJobResult
}

var TaskAgentPoolMaintenanceJobResultValues = taskAgentPoolMaintenanceJobResultValuesType{
	Succeeded: "succeeded",
	Failed:    "failed",
	Canceled:  "canceled",
}

type TaskAgentPoolMaintenanceJobStatus string

type taskAgentPoolMaintenanceJobStatusValuesType struct {
	InProgress TaskAgentPoolMaintenanceJobStatus
	Completed  TaskAgentPoolMaintenanceJobStatus
	Cancelling TaskAgentPoolMaintenanceJobStatus
	Queued     TaskAgentPoolMaintenanceJobStatus
}

var TaskAgentPoolMaintenanceJobStatusValues = taskAgentPoolMaintenanceJobStatusValuesType{
	InProgress: "inProgress",
	Completed:  "completed",
	Cancelling: "cancelling",
	Queued:     "queued",
}

type TaskAgentPoolMaintenanceJobTargetAgent struct {
	Agent  *TaskAgentReference                `json:"agent,omitempty"`
	JobId  *int                               `json:"jobId,omitempty"`
	Result *TaskAgentPoolMaintenanceJobResult `json:"result,omitempty"`
	Status *TaskAgentPoolMaintenanceJobStatus `json:"status,omitempty"`
}

type TaskAgentPoolMaintenanceOptions struct {
	// time to consider a System.DefaultWorkingDirectory is stale
	WorkingDirectoryExpirationInDays *int `json:"workingDirectoryExpirationInDays,omitempty"`
}

type TaskAgentPoolMaintenanceRetentionPolicy struct {
	// Number of records to keep for maintenance job executed with this definition.
	NumberOfHistoryRecordsToKeep *int `json:"numberOfHistoryRecordsToKeep,omitempty"`
}

type TaskAgentPoolMaintenanceSchedule struct {
	// Days for a build (flags enum for days of the week)
	DaysToBuild *TaskAgentPoolMaintenanceScheduleDays `json:"daysToBuild,omitempty"`
	// The Job Id of the Scheduled job that will queue the pool maintenance job.
	ScheduleJobId *uuid.UUID `json:"scheduleJobId,omitempty"`
	// Local timezone hour to start
	StartHours *int `json:"startHours,omitempty"`
	// Local timezone minute to start
	StartMinutes *int `json:"startMinutes,omitempty"`
	// Time zone of the build schedule (string representation of the time zone id)
	TimeZoneId *string `json:"timeZoneId,omitempty"`
}

type TaskAgentPoolMaintenanceScheduleDays string

type taskAgentPoolMaintenanceScheduleDaysValuesType struct {
	None      TaskAgentPoolMaintenanceScheduleDays
	Monday    TaskAgentPoolMaintenanceScheduleDays
	Tuesday   TaskAgentPoolMaintenanceScheduleDays
	Wednesday TaskAgentPoolMaintenanceScheduleDays
	Thursday  TaskAgentPoolMaintenanceScheduleDays
	Friday    TaskAgentPoolMaintenanceScheduleDays
	Saturday  TaskAgentPoolMaintenanceScheduleDays
	Sunday    TaskAgentPoolMaintenanceScheduleDays
	All       TaskAgentPoolMaintenanceScheduleDays
}

var TaskAgentPoolMaintenanceScheduleDaysValues = taskAgentPoolMaintenanceScheduleDaysValuesType{
	// Do not run.
	None: "none",
	// Run on Monday.
	Monday: "monday",
	// Run on Tuesday.
	Tuesday: "tuesday",
	// Run on Wednesday.
	Wednesday: "wednesday",
	// Run on Thursday.
	Thursday: "thursday",
	// Run on Friday.
	Friday: "friday",
	// Run on Saturday.
	Saturday: "saturday",
	// Run on Sunday.
	Sunday: "sunday",
	// Run on all days of the week.
	All: "all",
}

type TaskAgentPoolReference struct {
	Id *int `json:"id,omitempty"`
	// Gets or sets a value indicating whether or not this pool is managed by the service.
	IsHosted *bool `json:"isHosted,omitempty"`
	// Determines whether the pool is legacy.
	IsLegacy *bool   `json:"isLegacy,omitempty"`
	Name     *string `json:"name,omitempty"`
	// Gets or sets the type of the pool
	PoolType *TaskAgentPoolType `json:"poolType,omitempty"`
	Scope    *uuid.UUID         `json:"scope,omitempty"`
	// Gets the current size of the pool.
	Size *int `json:"size,omitempty"`
}

type TaskAgentPoolStatus struct {
	Id *int `json:"id,omitempty"`
	// Gets or sets a value indicating whether or not this pool is managed by the service.
	IsHosted *bool `json:"isHosted,omitempty"`
	// Determines whether the pool is legacy.
	IsLegacy *bool   `json:"isLegacy,omitempty"`
	Name     *string `json:"name,omitempty"`
	// Gets or sets the type of the pool
	PoolType *TaskAgentPoolType `json:"poolType,omitempty"`
	Scope    *uuid.UUID         `json:"scope,omitempty"`
	// Gets the current size of the pool.
	Size *int `json:"size,omitempty"`
	// Number of requests queued and assigned to an agent. Not running yet.
	AssignedRequestCount *int `json:"assignedRequestCount,omitempty"`
	// Number of queued requests which are not assigned to any agents
	QueuedRequestCount *int `json:"queuedRequestCount,omitempty"`
	// Number of currently running requests
	RunningRequestCount *int `json:"runningRequestCount,omitempty"`
}

type TaskAgentPoolSummary struct {
	ColumnsHeader    *MetricsColumnsHeader       `json:"columnsHeader,omitempty"`
	DeploymentGroups *[]DeploymentGroupReference `json:"deploymentGroups,omitempty"`
	Pool             *TaskAgentPoolReference     `json:"pool,omitempty"`
	Queues           *[]TaskAgentQueue           `json:"queues,omitempty"`
	Rows             *[]MetricsRow               `json:"rows,omitempty"`
}

// The type of agent pool.
type TaskAgentPoolType string

type taskAgentPoolTypeValuesType struct {
	Automation TaskAgentPoolType
	Deployment TaskAgentPoolType
}

var TaskAgentPoolTypeValues = taskAgentPoolTypeValuesType{
	// A typical pool of task agents
	Automation: "automation",
	// A deployment pool
	Deployment: "deployment",
}

// Represents the public key portion of an RSA asymmetric key.
type TaskAgentPublicKey struct {
	// Gets or sets the exponent for the public key.
	Exponent *[]byte `json:"exponent,omitempty"`
	// Gets or sets the modulus for the public key.
	Modulus *[]byte `json:"modulus,omitempty"`
}

// An agent queue.
type TaskAgentQueue struct {
	// ID of the queue
	Id *int `json:"id,omitempty"`
	// Name of the queue
	Name *string `json:"name,omitempty"`
	// Pool reference for this queue
	Pool *TaskAgentPoolReference `json:"pool,omitempty"`
	// Project ID
	ProjectId *uuid.UUID `json:"projectId,omitempty"`
}

// [Flags] Filters queues based on whether the calling user has permission to use or manage the queue.
type TaskAgentQueueActionFilter string

type taskAgentQueueActionFilterValuesType struct {
	None   TaskAgentQueueActionFilter
	Manage TaskAgentQueueActionFilter
	Use    TaskAgentQueueActionFilter
}

var TaskAgentQueueActionFilterValues = taskAgentQueueActionFilterValuesType{
	None:   "none",
	Manage: "manage",
	Use:    "use",
}

// A reference to an agent.
type TaskAgentReference struct {
	Links interface{} `json:"_links,omitempty"`
	// This agent's access point.
	AccessPoint *string `json:"accessPoint,omitempty"`
	// Whether or not this agent should run jobs.
	Enabled *bool `json:"enabled,omitempty"`
	// Identifier of the agent.
	Id *int `json:"id,omitempty"`
	// Name of the agent.
	Name *string `json:"name,omitempty"`
	// Agent OS.
	OsDescription *string `json:"osDescription,omitempty"`
	// Provisioning state of this agent.
	ProvisioningState *string `json:"provisioningState,omitempty"`
	// Whether or not the agent is online.
	Status *TaskAgentStatus `json:"status,omitempty"`
	// Agent version.
	Version *string `json:"version,omitempty"`
}

// Represents a session for performing message exchanges from an agent.
type TaskAgentSession struct {
	// Gets or sets the agent which is the target of the session.
	Agent *TaskAgentReference `json:"agent,omitempty"`
	// Gets the key used to encrypt message traffic for this session.
	EncryptionKey *TaskAgentSessionKey `json:"encryptionKey,omitempty"`
	// Gets or sets the owner name of this session. Generally this will be the machine of origination.
	OwnerName *string `json:"ownerName,omitempty"`
	// Gets the unique identifier for this session.
	SessionId          *uuid.UUID         `json:"sessionId,omitempty"`
	SystemCapabilities *map[string]string `json:"systemCapabilities,omitempty"`
}

// Represents a symmetric key used for message-level encryption for communication sent to an agent.
type TaskAgentSessionKey struct {
	// Gets or sets a value indicating whether or not the key value is encrypted. If this value is true, the Value property should be decrypted using the <c>RSA</c> key exchanged with the server during registration.
	Encrypted *bool `json:"encrypted,omitempty"`
	// Gets or sets the symmetric key value.
	Value *[]byte `json:"value,omitempty"`
}

type TaskAgentStatus string

type taskAgentStatusValuesType struct {
	Offline TaskAgentStatus
	Online  TaskAgentStatus
}

var TaskAgentStatusValues = taskAgentStatusValuesType{
	Offline: "offline",
	Online:  "online",
}

// [Flags] This is useful in getting a list of deployment targets, filtered by the deployment agent status.
type TaskAgentStatusFilter string

type taskAgentStatusFilterValuesType struct {
	Offline TaskAgentStatusFilter
	Online  TaskAgentStatusFilter
	All     TaskAgentStatusFilter
}

var TaskAgentStatusFilterValues = taskAgentStatusFilterValuesType{
	// Only deployment targets that are offline.
	Offline: "offline",
	// Only deployment targets that are online.
	Online: "online",
	// All deployment targets.
	All: "all",
}

// Details about an agent update.
type TaskAgentUpdate struct {
	// Current state of this agent update.
	CurrentState *string `json:"currentState,omitempty"`
	// Reason for this update.
	Reason *TaskAgentUpdateReason `json:"reason,omitempty"`
	// Identity which requested this update.
	RequestedBy *webapi.IdentityRef `json:"requestedBy,omitempty"`
	// Date on which this update was requested.
	RequestTime *azuredevops.Time `json:"requestTime,omitempty"`
	// Source agent version of the update.
	SourceVersion *PackageVersion `json:"sourceVersion,omitempty"`
	// Target agent version of the update.
	TargetVersion *PackageVersion `json:"targetVersion,omitempty"`
}

type TaskAgentUpdateReason struct {
	Code *TaskAgentUpdateReasonType `json:"code,omitempty"`
}

type TaskAgentUpdateReasonType string

type taskAgentUpdateReasonTypeValuesType struct {
	Manual                  TaskAgentUpdateReasonType
	MinAgentVersionRequired TaskAgentUpdateReasonType
}

var TaskAgentUpdateReasonTypeValues = taskAgentUpdateReasonTypeValuesType{
	Manual:                  "manual",
	MinAgentVersionRequired: "minAgentVersionRequired",
}

type TaskAssignedEvent struct {
	JobId  *uuid.UUID `json:"jobId,omitempty"`
	Name   *string    `json:"name,omitempty"`
	TaskId *uuid.UUID `json:"taskId,omitempty"`
}

type TaskAttachment struct {
	Links         interface{}       `json:"_links,omitempty"`
	CreatedOn     *azuredevops.Time `json:"createdOn,omitempty"`
	LastChangedBy *uuid.UUID        `json:"lastChangedBy,omitempty"`
	LastChangedOn *azuredevops.Time `json:"lastChangedOn,omitempty"`
	Name          *string           `json:"name,omitempty"`
	RecordId      *uuid.UUID        `json:"recordId,omitempty"`
	TimelineId    *uuid.UUID        `json:"timelineId,omitempty"`
	Type          *string           `json:"type,omitempty"`
}

type TaskCompletedEvent struct {
	JobId  *uuid.UUID  `json:"jobId,omitempty"`
	Name   *string     `json:"name,omitempty"`
	TaskId *uuid.UUID  `json:"taskId,omitempty"`
	Result *TaskResult `json:"result,omitempty"`
}

type TaskDefinition struct {
	AgentExecution         *TaskExecution       `json:"agentExecution,omitempty"`
	Author                 *string              `json:"author,omitempty"`
	Category               *string              `json:"category,omitempty"`
	ContentsUploaded       *bool                `json:"contentsUploaded,omitempty"`
	ContributionIdentifier *string              `json:"contributionIdentifier,omitempty"`
	ContributionVersion    *string              `json:"contributionVersion,omitempty"`
	DataSourceBindings     *[]DataSourceBinding `json:"dataSourceBindings,omitempty"`
	DefinitionType         *string              `json:"definitionType,omitempty"`
	Demands                *[]interface{}       `json:"demands,omitempty"`
	Deprecated             *bool                `json:"deprecated,omitempty"`
	Description            *string              `json:"description,omitempty"`
	Disabled               *bool                `json:"disabled,omitempty"`
	// Deprecated: Ecosystem property is not currently supported.
	Ecosystem                *string                 `json:"ecosystem,omitempty"`
	Execution                *map[string]interface{} `json:"execution,omitempty"`
	FriendlyName             *string                 `json:"friendlyName,omitempty"`
	Groups                   *[]TaskGroupDefinition  `json:"groups,omitempty"`
	HelpMarkDown             *string                 `json:"helpMarkDown,omitempty"`
	HelpUrl                  *string                 `json:"helpUrl,omitempty"`
	HostType                 *string                 `json:"hostType,omitempty"`
	IconUrl                  *string                 `json:"iconUrl,omitempty"`
	Id                       *uuid.UUID              `json:"id,omitempty"`
	Inputs                   *[]TaskInputDefinition  `json:"inputs,omitempty"`
	InstanceNameFormat       *string                 `json:"instanceNameFormat,omitempty"`
	MinimumAgentVersion      *string                 `json:"minimumAgentVersion,omitempty"`
	Name                     *string                 `json:"name,omitempty"`
	OutputVariables          *[]TaskOutputVariable   `json:"outputVariables,omitempty"`
	PackageLocation          *string                 `json:"packageLocation,omitempty"`
	PackageType              *string                 `json:"packageType,omitempty"`
	PostJobExecution         *map[string]interface{} `json:"postJobExecution,omitempty"`
	PreJobExecution          *map[string]interface{} `json:"preJobExecution,omitempty"`
	Preview                  *bool                   `json:"preview,omitempty"`
	ReleaseNotes             *string                 `json:"releaseNotes,omitempty"`
	RunsOn                   *[]string               `json:"runsOn,omitempty"`
	Satisfies                *[]string               `json:"satisfies,omitempty"`
	ServerOwned              *bool                   `json:"serverOwned,omitempty"`
	ShowEnvironmentVariables *bool                   `json:"showEnvironmentVariables,omitempty"`
	SourceDefinitions        *[]TaskSourceDefinition `json:"sourceDefinitions,omitempty"`
	SourceLocation           *string                 `json:"sourceLocation,omitempty"`
	Version                  *TaskVersion            `json:"version,omitempty"`
	Visibility               *[]string               `json:"visibility,omitempty"`
}

type TaskDefinitionEndpoint struct {
	// An ID that identifies a service connection to be used for authenticating endpoint requests.
	ConnectionId *string `json:"connectionId,omitempty"`
	// An Json based keyselector to filter response returned by fetching the endpoint <c>Url</c>.A Json based keyselector must be prefixed with "jsonpath:". KeySelector can be used to specify the filter to get the keys for the values specified with Selector. <example> The following keyselector defines an Json for extracting nodes named 'ServiceName'. <code> endpoint.KeySelector = "jsonpath://ServiceName"; </code></example>
	KeySelector *string `json:"keySelector,omitempty"`
	// The scope as understood by Connected Services. Essentially, a project-id for now.
	Scope *string `json:"scope,omitempty"`
	// An XPath/Json based selector to filter response returned by fetching the endpoint <c>Url</c>. An XPath based selector must be prefixed with the string "xpath:". A Json based selector must be prefixed with "jsonpath:". <example> The following selector defines an XPath for extracting nodes named 'ServiceName'. <code> endpoint.Selector = "xpath://ServiceName"; </code></example>
	Selector *string `json:"selector,omitempty"`
	// TaskId that this endpoint belongs to.
	TaskId *string `json:"taskId,omitempty"`
	// URL to GET.
	Url *string `json:"url,omitempty"`
}

type TaskDefinitionReference struct {
	// Gets or sets the definition type. Values can be 'task' or 'metaTask'.
	DefinitionType *string `json:"definitionType,omitempty"`
	// Gets or sets the unique identifier of task.
	Id *uuid.UUID `json:"id,omitempty"`
	// Gets or sets the version specification of task.
	VersionSpec *string `json:"versionSpec,omitempty"`
}

type TaskDefinitionStatus string

type taskDefinitionStatusValuesType struct {
	Preinstalled            TaskDefinitionStatus
	ReceivedInstallOrUpdate TaskDefinitionStatus
	Installed               TaskDefinitionStatus
	ReceivedUninstall       TaskDefinitionStatus
	Uninstalled             TaskDefinitionStatus
	RequestedUpdate         TaskDefinitionStatus
	Updated                 TaskDefinitionStatus
	AlreadyUpToDate         TaskDefinitionStatus
	InlineUpdateReceived    TaskDefinitionStatus
}

var TaskDefinitionStatusValues = taskDefinitionStatusValuesType{
	Preinstalled:            "preinstalled",
	ReceivedInstallOrUpdate: "receivedInstallOrUpdate",
	Installed:               "installed",
	ReceivedUninstall:       "receivedUninstall",
	Uninstalled:             "uninstalled",
	RequestedUpdate:         "requestedUpdate",
	Updated:                 "updated",
	AlreadyUpToDate:         "alreadyUpToDate",
	InlineUpdateReceived:    "inlineUpdateReceived",
}

type TaskEvent struct {
	JobId  *uuid.UUID `json:"jobId,omitempty"`
	Name   *string    `json:"name,omitempty"`
	TaskId *uuid.UUID `json:"taskId,omitempty"`
}

type TaskExecution struct {
	// The utility task to run.  Specifying this means that this task definition is simply a meta task to call another task. This is useful for tasks that call utility tasks like powershell and commandline
	ExecTask *TaskReference `json:"execTask,omitempty"`
	// If a task is going to run code, then this provides the type/script etc... information by platform. For example, it might look like. net45: { typeName: "Microsoft.TeamFoundation.Automation.Tasks.PowerShellTask", assemblyName: "Microsoft.TeamFoundation.Automation.Tasks.PowerShell.dll" } net20: { typeName: "Microsoft.TeamFoundation.Automation.Tasks.PowerShellTask", assemblyName: "Microsoft.TeamFoundation.Automation.Tasks.PowerShell.dll" } java: { jar: "powershelltask.tasks.automation.teamfoundation.microsoft.com", } node: { script: "powershellhost.js", }
	PlatformInstructions *map[string]map[string]string `json:"platformInstructions,omitempty"`
}

type TaskGroup struct {
	AgentExecution           *TaskExecution          `json:"agentExecution,omitempty"`
	Author                   *string                 `json:"author,omitempty"`
	Category                 *string                 `json:"category,omitempty"`
	ContentsUploaded         *bool                   `json:"contentsUploaded,omitempty"`
	ContributionIdentifier   *string                 `json:"contributionIdentifier,omitempty"`
	ContributionVersion      *string                 `json:"contributionVersion,omitempty"`
	DataSourceBindings       *[]DataSourceBinding    `json:"dataSourceBindings,omitempty"`
	DefinitionType           *string                 `json:"definitionType,omitempty"`
	Demands                  *[]interface{}          `json:"demands,omitempty"`
	Deprecated               *bool                   `json:"deprecated,omitempty"`
	Description              *string                 `json:"description,omitempty"`
	Disabled                 *bool                   `json:"disabled,omitempty"`
	Ecosystem                *string                 `json:"ecosystem,omitempty"`
	Execution                *map[string]interface{} `json:"execution,omitempty"`
	FriendlyName             *string                 `json:"friendlyName,omitempty"`
	Groups                   *[]TaskGroupDefinition  `json:"groups,omitempty"`
	HelpMarkDown             *string                 `json:"helpMarkDown,omitempty"`
	HelpUrl                  *string                 `json:"helpUrl,omitempty"`
	HostType                 *string                 `json:"hostType,omitempty"`
	IconUrl                  *string                 `json:"iconUrl,omitempty"`
	Id                       *uuid.UUID              `json:"id,omitempty"`
	Inputs                   *[]TaskInputDefinition  `json:"inputs,omitempty"`
	InstanceNameFormat       *string                 `json:"instanceNameFormat,omitempty"`
	MinimumAgentVersion      *string                 `json:"minimumAgentVersion,omitempty"`
	Name                     *string                 `json:"name,omitempty"`
	OutputVariables          *[]TaskOutputVariable   `json:"outputVariables,omitempty"`
	PackageLocation          *string                 `json:"packageLocation,omitempty"`
	PackageType              *string                 `json:"packageType,omitempty"`
	PostJobExecution         *map[string]interface{} `json:"postJobExecution,omitempty"`
	PreJobExecution          *map[string]interface{} `json:"preJobExecution,omitempty"`
	Preview                  *bool                   `json:"preview,omitempty"`
	ReleaseNotes             *string                 `json:"releaseNotes,omitempty"`
	RunsOn                   *[]string               `json:"runsOn,omitempty"`
	Satisfies                *[]string               `json:"satisfies,omitempty"`
	ServerOwned              *bool                   `json:"serverOwned,omitempty"`
	ShowEnvironmentVariables *bool                   `json:"showEnvironmentVariables,omitempty"`
	SourceDefinitions        *[]TaskSourceDefinition `json:"sourceDefinitions,omitempty"`
	SourceLocation           *string                 `json:"sourceLocation,omitempty"`
	Version                  *TaskVersion            `json:"version,omitempty"`
	Visibility               *[]string               `json:"visibility,omitempty"`
	// Gets or sets comment.
	Comment *string `json:"comment,omitempty"`
	// Gets or sets the identity who created.
	CreatedBy *webapi.IdentityRef `json:"createdBy,omitempty"`
	// Gets or sets date on which it got created.
	CreatedOn *azuredevops.Time `json:"createdOn,omitempty"`
	// Gets or sets as 'true' to indicate as deleted, 'false' otherwise.
	Deleted *bool `json:"deleted,omitempty"`
	// Gets or sets the identity who modified.
	ModifiedBy *webapi.IdentityRef `json:"modifiedBy,omitempty"`
	// Gets or sets date on which it got modified.
	ModifiedOn *azuredevops.Time `json:"modifiedOn,omitempty"`
	// Gets or sets the owner.
	Owner *string `json:"owner,omitempty"`
	// Gets or sets parent task group Id. This is used while creating a draft task group.
	ParentDefinitionId *uuid.UUID `json:"parentDefinitionId,omitempty"`
	// Gets or sets revision.
	Revision *int `json:"revision,omitempty"`
	// Gets or sets the tasks.
	Tasks *[]TaskGroupStep `json:"tasks,omitempty"`
}

type TaskGroupCreateParameter struct {
	// Sets author name of the task group.
	Author *string `json:"author,omitempty"`
	// Sets category of the task group.
	Category *string `json:"category,omitempty"`
	// Sets description of the task group.
	Description *string `json:"description,omitempty"`
	// Sets friendly name of the task group.
	FriendlyName *string `json:"friendlyName,omitempty"`
	// Sets url icon of the task group.
	IconUrl *string `json:"iconUrl,omitempty"`
	// Sets input for the task group.
	Inputs *[]TaskInputDefinition `json:"inputs,omitempty"`
	// Sets display name of the task group.
	InstanceNameFormat *string `json:"instanceNameFormat,omitempty"`
	// Sets name of the task group.
	Name *string `json:"name,omitempty"`
	// Sets parent task group Id. This is used while creating a draft task group.
	ParentDefinitionId *uuid.UUID `json:"parentDefinitionId,omitempty"`
	// Sets RunsOn of the task group. Value can be 'Agent', 'Server' or 'DeploymentGroup'.
	RunsOn *[]string `json:"runsOn,omitempty"`
	// Sets tasks for the task group.
	Tasks *[]TaskGroupStep `json:"tasks,omitempty"`
	// Sets version of the task group.
	Version *TaskVersion `json:"version,omitempty"`
}

type TaskGroupDefinition struct {
	DisplayName *string   `json:"displayName,omitempty"`
	IsExpanded  *bool     `json:"isExpanded,omitempty"`
	Name        *string   `json:"name,omitempty"`
	Tags        *[]string `json:"tags,omitempty"`
	VisibleRule *string   `json:"visibleRule,omitempty"`
}

// [Flags]
type TaskGroupExpands string

type taskGroupExpandsValuesType struct {
	None  TaskGroupExpands
	Tasks TaskGroupExpands
}

var TaskGroupExpandsValues = taskGroupExpandsValuesType{
	None:  "none",
	Tasks: "tasks",
}

// Specifies the desired ordering of taskGroups.
type TaskGroupQueryOrder string

type taskGroupQueryOrderValuesType struct {
	CreatedOnAscending  TaskGroupQueryOrder
	CreatedOnDescending TaskGroupQueryOrder
}

var TaskGroupQueryOrderValues = taskGroupQueryOrderValuesType{
	// Order by createdon ascending.
	CreatedOnAscending: "createdOnAscending",
	// Order by createdon descending.
	CreatedOnDescending: "createdOnDescending",
}

type TaskGroupRevision struct {
	ChangedBy    *webapi.IdentityRef `json:"changedBy,omitempty"`
	ChangedDate  *azuredevops.Time   `json:"changedDate,omitempty"`
	ChangeType   *AuditAction        `json:"changeType,omitempty"`
	Comment      *string             `json:"comment,omitempty"`
	FileId       *int                `json:"fileId,omitempty"`
	MajorVersion *int                `json:"majorVersion,omitempty"`
	Revision     *int                `json:"revision,omitempty"`
	TaskGroupId  *uuid.UUID          `json:"taskGroupId,omitempty"`
}

// Represents tasks in the task group.
type TaskGroupStep struct {
	// Gets or sets as 'true' to run the task always, 'false' otherwise.
	AlwaysRun *bool `json:"alwaysRun,omitempty"`
	// Gets or sets condition for the task.
	Condition *string `json:"condition,omitempty"`
	// Gets or sets as 'true' to continue on error, 'false' otherwise.
	ContinueOnError *bool `json:"continueOnError,omitempty"`
	// Gets or sets the display name.
	DisplayName *string `json:"displayName,omitempty"`
	// Gets or sets as task is enabled or not.
	Enabled *bool `json:"enabled,omitempty"`
	// Gets dictionary of environment variables.
	Environment *map[string]string `json:"environment,omitempty"`
	// Gets or sets dictionary of inputs.
	Inputs *map[string]string `json:"inputs,omitempty"`
	// Gets or sets the reference of the task.
	Task *TaskDefinitionReference `json:"task,omitempty"`
	// Gets or sets the maximum time, in minutes, that a task is allowed to execute on agent before being cancelled by server. A zero value indicates an infinite timeout.
	TimeoutInMinutes *int `json:"timeoutInMinutes,omitempty"`
}

type TaskGroupUpdateParameter struct {
	// Sets author name of the task group.
	Author *string `json:"author,omitempty"`
	// Sets category of the task group.
	Category *string `json:"category,omitempty"`
	// Sets comment of the task group.
	Comment *string `json:"comment,omitempty"`
	// Sets description of the task group.
	Description *string `json:"description,omitempty"`
	// Sets friendly name of the task group.
	FriendlyName *string `json:"friendlyName,omitempty"`
	// Sets url icon of the task group.
	IconUrl *string `json:"iconUrl,omitempty"`
	// Sets the unique identifier of this field.
	Id *uuid.UUID `json:"id,omitempty"`
	// Sets input for the task group.
	Inputs *[]TaskInputDefinition `json:"inputs,omitempty"`
	// Sets display name of the task group.
	InstanceNameFormat *string `json:"instanceNameFormat,omitempty"`
	// Sets name of the task group.
	Name *string `json:"name,omitempty"`
	// Gets or sets parent task group Id. This is used while creating a draft task group.
	ParentDefinitionId *uuid.UUID `json:"parentDefinitionId,omitempty"`
	// Sets revision of the task group.
	Revision *int `json:"revision,omitempty"`
	// Sets RunsOn of the task group. Value can be 'Agent', 'Server' or 'DeploymentGroup'.
	RunsOn *[]string `json:"runsOn,omitempty"`
	// Sets tasks for the task group.
	Tasks *[]TaskGroupStep `json:"tasks,omitempty"`
	// Sets version of the task group.
	Version *TaskVersion `json:"version,omitempty"`
}

type TaskHubLicenseDetails struct {
	EnterpriseUsersCount               *int                           `json:"enterpriseUsersCount,omitempty"`
	FailedToReachAllProviders          *bool                          `json:"failedToReachAllProviders,omitempty"`
	FreeHostedLicenseCount             *int                           `json:"freeHostedLicenseCount,omitempty"`
	FreeLicenseCount                   *int                           `json:"freeLicenseCount,omitempty"`
	HasLicenseCountEverUpdated         *bool                          `json:"hasLicenseCountEverUpdated,omitempty"`
	HostedAgentMinutesFreeCount        *int                           `json:"hostedAgentMinutesFreeCount,omitempty"`
	HostedAgentMinutesUsedCount        *int                           `json:"hostedAgentMinutesUsedCount,omitempty"`
	HostedLicensesArePremium           *bool                          `json:"hostedLicensesArePremium,omitempty"`
	MarketplacePurchasedHostedLicenses *[]MarketplacePurchasedLicense `json:"marketplacePurchasedHostedLicenses,omitempty"`
	MsdnUsersCount                     *int                           `json:"msdnUsersCount,omitempty"`
	// Microsoft-hosted licenses purchased from VSTS directly.
	PurchasedHostedLicenseCount *int `json:"purchasedHostedLicenseCount,omitempty"`
	// Self-hosted licenses purchased from VSTS directly.
	PurchasedLicenseCount    *int `json:"purchasedLicenseCount,omitempty"`
	TotalHostedLicenseCount  *int `json:"totalHostedLicenseCount,omitempty"`
	TotalLicenseCount        *int `json:"totalLicenseCount,omitempty"`
	TotalPrivateLicenseCount *int `json:"totalPrivateLicenseCount,omitempty"`
}

type TaskInputDefinition struct {
	Aliases      *[]string                                  `json:"aliases,omitempty"`
	DefaultValue *string                                    `json:"defaultValue,omitempty"`
	GroupName    *string                                    `json:"groupName,omitempty"`
	HelpMarkDown *string                                    `json:"helpMarkDown,omitempty"`
	Label        *string                                    `json:"label,omitempty"`
	Name         *string                                    `json:"name,omitempty"`
	Options      *map[string]string                         `json:"options,omitempty"`
	Properties   *map[string]string                         `json:"properties,omitempty"`
	Required     *bool                                      `json:"required,omitempty"`
	Type         *string                                    `json:"type,omitempty"`
	Validation   *distributedtaskcommon.TaskInputValidation `json:"validation,omitempty"`
	VisibleRule  *string                                    `json:"visibleRule,omitempty"`
}

type TaskInstance struct {
	Id               *uuid.UUID         `json:"id,omitempty"`
	Inputs           *map[string]string `json:"inputs,omitempty"`
	Name             *string            `json:"name,omitempty"`
	Version          *string            `json:"version,omitempty"`
	AlwaysRun        *bool              `json:"alwaysRun,omitempty"`
	Condition        *string            `json:"condition,omitempty"`
	ContinueOnError  *bool              `json:"continueOnError,omitempty"`
	DisplayName      *string            `json:"displayName,omitempty"`
	Enabled          *bool              `json:"enabled,omitempty"`
	Environment      *map[string]string `json:"environment,omitempty"`
	InstanceId       *uuid.UUID         `json:"instanceId,omitempty"`
	RefName          *string            `json:"refName,omitempty"`
	TimeoutInMinutes *int               `json:"timeoutInMinutes,omitempty"`
}

type TaskLog struct {
	Id            *int              `json:"id,omitempty"`
	Location      *string           `json:"location,omitempty"`
	CreatedOn     *azuredevops.Time `json:"createdOn,omitempty"`
	IndexLocation *string           `json:"indexLocation,omitempty"`
	LastChangedOn *azuredevops.Time `json:"lastChangedOn,omitempty"`
	LineCount     *uint64           `json:"lineCount,omitempty"`
	Path          *string           `json:"path,omitempty"`
}

type TaskLogReference struct {
	Id       *int    `json:"id,omitempty"`
	Location *string `json:"location,omitempty"`
}

type TaskOrchestrationContainer struct {
	ItemType        *TaskOrchestrationItemType  `json:"itemType,omitempty"`
	Children        *[]TaskOrchestrationItem    `json:"children,omitempty"`
	ContinueOnError *bool                       `json:"continueOnError,omitempty"`
	Data            *map[string]string          `json:"data,omitempty"`
	MaxConcurrency  *int                        `json:"maxConcurrency,omitempty"`
	Parallel        *bool                       `json:"parallel,omitempty"`
	Rollback        *TaskOrchestrationContainer `json:"rollback,omitempty"`
}

type TaskOrchestrationItem struct {
	ItemType *TaskOrchestrationItemType `json:"itemType,omitempty"`
}

type TaskOrchestrationItemType string

type taskOrchestrationItemTypeValuesType struct {
	Container TaskOrchestrationItemType
	Job       TaskOrchestrationItemType
}

var TaskOrchestrationItemTypeValues = taskOrchestrationItemTypeValuesType{
	Container: "container",
	Job:       "job",
}

type TaskOrchestrationJob struct {
	ItemType         *TaskOrchestrationItemType `json:"itemType,omitempty"`
	Demands          *[]interface{}             `json:"demands,omitempty"`
	ExecuteAs        *webapi.IdentityRef        `json:"executeAs,omitempty"`
	ExecutionMode    *string                    `json:"executionMode,omitempty"`
	ExecutionTimeout interface{}                `json:"executionTimeout,omitempty"`
	InstanceId       *uuid.UUID                 `json:"instanceId,omitempty"`
	Name             *string                    `json:"name,omitempty"`
	RefName          *string                    `json:"refName,omitempty"`
	Tasks            *[]TaskInstance            `json:"tasks,omitempty"`
	Variables        *map[string]string         `json:"variables,omitempty"`
}

type TaskOrchestrationOwner struct {
	Links interface{} `json:"_links,omitempty"`
	Id    *int        `json:"id,omitempty"`
	Name  *string     `json:"name,omitempty"`
}

type TaskOrchestrationPlan struct {
	ArtifactLocation  *string                     `json:"artifactLocation,omitempty"`
	ArtifactUri       *string                     `json:"artifactUri,omitempty"`
	Definition        *TaskOrchestrationOwner     `json:"definition,omitempty"`
	Owner             *TaskOrchestrationOwner     `json:"owner,omitempty"`
	PlanGroup         *string                     `json:"planGroup,omitempty"`
	PlanId            *uuid.UUID                  `json:"planId,omitempty"`
	PlanType          *string                     `json:"planType,omitempty"`
	ScopeIdentifier   *uuid.UUID                  `json:"scopeIdentifier,omitempty"`
	Version           *int                        `json:"version,omitempty"`
	Environment       *PlanEnvironment            `json:"environment,omitempty"`
	FinishTime        *azuredevops.Time           `json:"finishTime,omitempty"`
	Implementation    *TaskOrchestrationContainer `json:"implementation,omitempty"`
	InitializationLog *TaskLogReference           `json:"initializationLog,omitempty"`
	RequestedById     *uuid.UUID                  `json:"requestedById,omitempty"`
	RequestedForId    *uuid.UUID                  `json:"requestedForId,omitempty"`
	Result            *TaskResult                 `json:"result,omitempty"`
	ResultCode        *string                     `json:"resultCode,omitempty"`
	StartTime         *azuredevops.Time           `json:"startTime,omitempty"`
	State             *TaskOrchestrationPlanState `json:"state,omitempty"`
	Timeline          *TimelineReference          `json:"timeline,omitempty"`
}

type TaskOrchestrationPlanGroup struct {
	PlanGroup       *string                `json:"planGroup,omitempty"`
	Project         *ProjectReference      `json:"project,omitempty"`
	RunningRequests *[]TaskAgentJobRequest `json:"runningRequests,omitempty"`
}

type TaskOrchestrationPlanGroupsQueueMetrics struct {
	Count  *int             `json:"count,omitempty"`
	Status *PlanGroupStatus `json:"status,omitempty"`
}

type TaskOrchestrationPlanReference struct {
	ArtifactLocation *string                 `json:"artifactLocation,omitempty"`
	ArtifactUri      *string                 `json:"artifactUri,omitempty"`
	Definition       *TaskOrchestrationOwner `json:"definition,omitempty"`
	Owner            *TaskOrchestrationOwner `json:"owner,omitempty"`
	PlanGroup        *string                 `json:"planGroup,omitempty"`
	PlanId           *uuid.UUID              `json:"planId,omitempty"`
	PlanType         *string                 `json:"planType,omitempty"`
	ScopeIdentifier  *uuid.UUID              `json:"scopeIdentifier,omitempty"`
	Version          *int                    `json:"version,omitempty"`
}

type TaskOrchestrationPlanState string

type taskOrchestrationPlanStateValuesType struct {
	InProgress TaskOrchestrationPlanState
	Queued     TaskOrchestrationPlanState
	Completed  TaskOrchestrationPlanState
	Throttled  TaskOrchestrationPlanState
}

var TaskOrchestrationPlanStateValues = taskOrchestrationPlanStateValuesType{
	InProgress: "inProgress",
	Queued:     "queued",
	Completed:  "completed",
	Throttled:  "throttled",
}

type TaskOrchestrationQueuedPlan struct {
	AssignTime      *azuredevops.Time       `json:"assignTime,omitempty"`
	Definition      *TaskOrchestrationOwner `json:"definition,omitempty"`
	Owner           *TaskOrchestrationOwner `json:"owner,omitempty"`
	PlanGroup       *string                 `json:"planGroup,omitempty"`
	PlanId          *uuid.UUID              `json:"planId,omitempty"`
	PoolId          *int                    `json:"poolId,omitempty"`
	QueuePosition   *int                    `json:"queuePosition,omitempty"`
	QueueTime       *azuredevops.Time       `json:"queueTime,omitempty"`
	ScopeIdentifier *uuid.UUID              `json:"scopeIdentifier,omitempty"`
}

type TaskOrchestrationQueuedPlanGroup struct {
	Definition    *TaskOrchestrationOwner        `json:"definition,omitempty"`
	Owner         *TaskOrchestrationOwner        `json:"owner,omitempty"`
	PlanGroup     *string                        `json:"planGroup,omitempty"`
	Plans         *[]TaskOrchestrationQueuedPlan `json:"plans,omitempty"`
	Project       *ProjectReference              `json:"project,omitempty"`
	QueuePosition *int                           `json:"queuePosition,omitempty"`
}

type TaskOutputVariable struct {
	Description *string `json:"description,omitempty"`
	Name        *string `json:"name,omitempty"`
}

type TaskPackageMetadata struct {
	// Gets the name of the package.
	Type *string `json:"type,omitempty"`
	// Gets the url of the package.
	Url *string `json:"url,omitempty"`
	// Gets the version of the package.
	Version *string `json:"version,omitempty"`
}

type TaskReference struct {
	Id      *uuid.UUID         `json:"id,omitempty"`
	Inputs  *map[string]string `json:"inputs,omitempty"`
	Name    *string            `json:"name,omitempty"`
	Version *string            `json:"version,omitempty"`
}

type TaskResult string

type taskResultValuesType struct {
	Succeeded           TaskResult
	SucceededWithIssues TaskResult
	Failed              TaskResult
	Canceled            TaskResult
	Skipped             TaskResult
	Abandoned           TaskResult
}

var TaskResultValues = taskResultValuesType{
	Succeeded:           "succeeded",
	SucceededWithIssues: "succeededWithIssues",
	Failed:              "failed",
	Canceled:            "canceled",
	Skipped:             "skipped",
	Abandoned:           "abandoned",
}

type TaskSourceDefinition struct {
	AuthKey     *string `json:"authKey,omitempty"`
	Endpoint    *string `json:"endpoint,omitempty"`
	KeySelector *string `json:"keySelector,omitempty"`
	Selector    *string `json:"selector,omitempty"`
	Target      *string `json:"target,omitempty"`
}

type TaskStartedEvent struct {
	JobId  *uuid.UUID `json:"jobId,omitempty"`
	Name   *string    `json:"name,omitempty"`
	TaskId *uuid.UUID `json:"taskId,omitempty"`
}

type TaskVersion struct {
	IsTest *bool `json:"isTest,omitempty"`
	Major  *int  `json:"major,omitempty"`
	Minor  *int  `json:"minor,omitempty"`
	Patch  *int  `json:"patch,omitempty"`
}

type Timeline struct {
	ChangeId      *int              `json:"changeId,omitempty"`
	Id            *uuid.UUID        `json:"id,omitempty"`
	Location      *string           `json:"location,omitempty"`
	LastChangedBy *uuid.UUID        `json:"lastChangedBy,omitempty"`
	LastChangedOn *azuredevops.Time `json:"lastChangedOn,omitempty"`
	Records       *[]TimelineRecord `json:"records,omitempty"`
}

type TimelineAttempt struct {
	// Gets or sets the attempt of the record.
	Attempt *int `json:"attempt,omitempty"`
	// Gets or sets the unique identifier for the record.
	Identifier *string `json:"identifier,omitempty"`
	// Gets or sets the record identifier located within the specified timeline.
	RecordId *uuid.UUID `json:"recordId,omitempty"`
	// Gets or sets the timeline identifier which owns the record representing this attempt.
	TimelineId *uuid.UUID `json:"timelineId,omitempty"`
}

type TimelineRecord struct {
	Attempt          *int                    `json:"attempt,omitempty"`
	ChangeId         *int                    `json:"changeId,omitempty"`
	CurrentOperation *string                 `json:"currentOperation,omitempty"`
	Details          *TimelineReference      `json:"details,omitempty"`
	ErrorCount       *int                    `json:"errorCount,omitempty"`
	FinishTime       *azuredevops.Time       `json:"finishTime,omitempty"`
	Id               *uuid.UUID              `json:"id,omitempty"`
	Identifier       *string                 `json:"identifier,omitempty"`
	Issues           *[]Issue                `json:"issues,omitempty"`
	LastModified     *azuredevops.Time       `json:"lastModified,omitempty"`
	Location         *string                 `json:"location,omitempty"`
	Log              *TaskLogReference       `json:"log,omitempty"`
	Name             *string                 `json:"name,omitempty"`
	Order            *int                    `json:"order,omitempty"`
	ParentId         *uuid.UUID              `json:"parentId,omitempty"`
	PercentComplete  *int                    `json:"percentComplete,omitempty"`
	PreviousAttempts *[]TimelineAttempt      `json:"previousAttempts,omitempty"`
	RefName          *string                 `json:"refName,omitempty"`
	Result           *TaskResult             `json:"result,omitempty"`
	ResultCode       *string                 `json:"resultCode,omitempty"`
	StartTime        *azuredevops.Time       `json:"startTime,omitempty"`
	State            *TimelineRecordState    `json:"state,omitempty"`
	Task             *TaskReference          `json:"task,omitempty"`
	Type             *string                 `json:"type,omitempty"`
	Variables        *map[string]interface{} `json:"variables,omitempty"`
	WarningCount     *int                    `json:"warningCount,omitempty"`
	WorkerName       *string                 `json:"workerName,omitempty"`
}

type TimelineRecordFeedLinesWrapper struct {
	Count  *int       `json:"count,omitempty"`
	StepId *uuid.UUID `json:"stepId,omitempty"`
	Value  *[]string  `json:"value,omitempty"`
}

type TimelineRecordState string

type timelineRecordStateValuesType struct {
	Pending    TimelineRecordState
	InProgress TimelineRecordState
	Completed  TimelineRecordState
}

var TimelineRecordStateValues = timelineRecordStateValuesType{
	Pending:    "pending",
	InProgress: "inProgress",
	Completed:  "completed",
}

type TimelineReference struct {
	ChangeId *int       `json:"changeId,omitempty"`
	Id       *uuid.UUID `json:"id,omitempty"`
	Location *string    `json:"location,omitempty"`
}

type ValidationItem struct {
	// Tells whether the current input is valid or not
	IsValid *bool `json:"isValid,omitempty"`
	// Reason for input validation failure
	Reason *string `json:"reason,omitempty"`
	// Type of validation item
	Type *string `json:"type,omitempty"`
	// Value to validate. The conditional expression to validate for the input for "expression" type Eg:eq(variables['Build.SourceBranch'], 'refs/heads/master');eq(value, 'refs/heads/master')
	Value *string `json:"value,omitempty"`
}

// A variable group is a collection of related variables.
type VariableGroup struct {
	// Gets or sets the identity who created the variable group.
	CreatedBy *webapi.IdentityRef `json:"createdBy,omitempty"`
	// Gets or sets the time when variable group was created.
	CreatedOn *azuredevops.Time `json:"createdOn,omitempty"`
	// Gets or sets description of the variable group.
	Description *string `json:"description,omitempty"`
	// Gets or sets id of the variable group.
	Id *int `json:"id,omitempty"`
	// Indicates whether variable group is shared with other projects or not.
	IsShared *bool `json:"isShared,omitempty"`
	// Gets or sets the identity who modified the variable group.
	ModifiedBy *webapi.IdentityRef `json:"modifiedBy,omitempty"`
	// Gets or sets the time when variable group was modified
	ModifiedOn *azuredevops.Time `json:"modifiedOn,omitempty"`
	// Gets or sets name of the variable group.
	Name *string `json:"name,omitempty"`
	// Gets or sets provider data.
	ProviderData interface{} `json:"providerData,omitempty"`
	// Gets or sets type of the variable group.
	Type *string `json:"type,omitempty"`
	// Gets or sets variables contained in the variable group.
	Variables *map[string]interface{} `json:"variables,omitempty"`
}

// [Flags]
type VariableGroupActionFilter string

type variableGroupActionFilterValuesType struct {
	None   VariableGroupActionFilter
	Manage VariableGroupActionFilter
	Use    VariableGroupActionFilter
}

var VariableGroupActionFilterValues = variableGroupActionFilterValuesType{
	None:   "none",
	Manage: "manage",
	Use:    "use",
}

type VariableGroupParameters struct {
	// Sets description of the variable group.
	Description *string `json:"description,omitempty"`
	// Sets name of the variable group.
	Name *string `json:"name,omitempty"`
	// Sets provider data.
	ProviderData interface{} `json:"providerData,omitempty"`
	// Sets type of the variable group.
	Type *string `json:"type,omitempty"`
	// Sets variables contained in the variable group.
	Variables *map[string]interface{} `json:"variables,omitempty"`
}

// Defines provider data of the variable group.
type VariableGroupProviderData struct {
}

// Specifies the desired ordering of variableGroups.
type VariableGroupQueryOrder string

type variableGroupQueryOrderValuesType struct {
	IdAscending  VariableGroupQueryOrder
	IdDescending VariableGroupQueryOrder
}

var VariableGroupQueryOrderValues = variableGroupQueryOrderValuesType{
	// Order by id ascending.
	IdAscending: "idAscending",
	// Order by id descending.
	IdDescending: "idDescending",
}

type VariableValue struct {
	IsSecret *bool   `json:"isSecret,omitempty"`
	Value    *string `json:"value,omitempty"`
}

type VirtualMachine struct {
	Agent *TaskAgent `json:"agent,omitempty"`
	Id    *int       `json:"id,omitempty"`
	Tags  *[]string  `json:"tags,omitempty"`
}

type VirtualMachineGroup struct {
	CreatedBy            *webapi.IdentityRef   `json:"createdBy,omitempty"`
	CreatedOn            *azuredevops.Time     `json:"createdOn,omitempty"`
	EnvironmentReference *EnvironmentReference `json:"environmentReference,omitempty"`
	Id                   *int                  `json:"id,omitempty"`
	LastModifiedBy       *webapi.IdentityRef   `json:"lastModifiedBy,omitempty"`
	LastModifiedOn       *azuredevops.Time     `json:"lastModifiedOn,omitempty"`
	Name                 *string               `json:"name,omitempty"`
	// Environment resource type
	Type   *EnvironmentResourceType `json:"type,omitempty"`
	PoolId *int                     `json:"poolId,omitempty"`
}

type VirtualMachineGroupCreateParameters struct {
	Name *string `json:"name,omitempty"`
}

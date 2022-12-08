package gophercloud

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

//Define Service error information.
const (
	//ECS
	EcsAuthRequired          = "Authentication required"
	EcsPoilcyNotAllow        = "Policy doesn't allow .*. to be performed"
	EcsTokenRoleEmpty        = "token role is empty, forbidden to perform this action"
	EcsTokenRoleForbidden    = "token role * is forbidden to perform this action"
	EcsErrorRoleToPerform    = "do not have the required roles, forbbiden to perform this action"
	EcsQuotaExceeded         = "Quota exceeded for instances"
	EcsPortNumberExceed      = "Maximum number of ports exceeded"
	EcsVolumeNumberOver      = "Volume number is over limit"
	EcsBlockImageNotFound    = "Block Device Mapping is Invalid: failed to get image.*."
	EcsImageNotFound         = "Image * could not be found."
	EcsFlavorNotFound        = "Flavor .*. could not be found"
	EcsNetworkNotFound       = "Network.*.could not be found"
	EcsBlockDevInvalid       = "Block Device Mapping is Invalid"
	EcsAZUnavailable         = "The requested availability zone is not available"
	EcsSecurityGroupNotFound = "Security group .*. not found"
	EcsKeyPairNotFound       = "Keypair .*. not found for user *"
	EcsInstanceGroupNotFound = "Instance group .*. could not be found"
	EcsInvalidMetadata       = "Invalid metadata.*"
	EcsUserDataBase64        = "User data needs to be valid base 64"
	EcsUserDataTooLarge      = "User data too large. User data must be no larger than .*"
	EcsInstanceDiskExceed    = "The created instance's disk would be too small"
	EcsFlavorMemoryNotEnough = "Flavor's memory is too small for requested image"
	EcsInstanceNotFound      = "Instance .*. could not be found"
	EcsInstanceIsLocked      = "Instance .*. is locked"
	EcsInstCantBeOperated    = "Cannot .*. instance .*. while it is in .*"
	EcsUnexpectedApiERROR    = "Unexpected API Error"
	EcsServerCantComply      = "The server could not comply with the request since it is either malformed.*."
	EcsInvalidFlavorRef      = "Invalid flavorRef provided"
	EcsInvalidKeyName        = "Invalid key_name provided"
	EcsInvalidInputField     = "Invalid input for field/attribute"
	EcsResourceSoldOut       = "Instance resource is temporarily sold out."

	//IMS
	Ims0027NoImageFoundWithId = "No image found with ID"
	Ims0144FailedFindImage    = "Failed to find image"

	//ELB
	ELB2001AdminStateUpFalse    = "Admin_state_up is not allowed with False"
	ELB2002IpNotValid           = "IP address .*. is not a valid IP for the specified subnet."
	ELB2003PoolNotFound         = "pool .*. could not be found"
	ELB2004MemberNotSupportPort = "Member not support protocol port *"
	ELB2005SubnetMismatch       = "Router of member's subnet .*. and router of loadbalancer's subnet .*. mismatch"
	ELB2006IpPortAlreadyPresent = "Member with address .*. and protocol_port .*. already present in pool .*."
	ELB2007MemberNotFound       = "member .*. could not be found"

	ELB6101QuotaExceeded  = "Quota exceeded for resources: \\['listener'\\]"
	ELB2541QuotaExceeded = "Quota exceeded for resources: \\['pool'\\]"
	ELBb015QuotaExceeded = "Quota exceeded for resources: \\['l7policy'\\]"
	ELB1071QuotaExceeded = "Quota exceeded for resources: \\['loadbalancer'\\]"
)

//Common Error.
const (
	//Com1000
	CE_MissingInputCode    = "Com.1000" //client error
	CE_MissingInputMessage = "Missing input for argument [%s]"

	//Com1001
	CE_StreamControlApiCode    = "Com.1001" //server error
	CE_StreamControlApiMessage = "The maximum request receiving rate is exceeded"

	//Com1002
	CE_InvalidInputCode    = "Com.1002" //client error
	CE_InvalidInputMessage = "Invalid input provided for argument [%s]"

	CE_OptionTypeNotStructCode    = "Com.1002" //client error
	CE_OptionTypeNotStructMessage = "Options type is not a struct"

	//Com1003
	CE_ResourceNotFoundCode    = "Com.1003" //client error
	CE_ResourceNotFoundMessage = "Unable to find %s with name %s"

	CE_MultipleResourcesFoundCode    = "Com.1003" //client error
	CE_MultipleResourcesFoundMessage = "Found %d %ss matching %s"

	CE_ErrUnexpectedTypeCode    = "Com.1003" //client error
	CE_ErrUnexpectedTypeMessage = "Expected %s but got %s"

	//Com1004
	CE_NoClientProvidedCode    = "Com.1004" //client error
	CE_NoClientProvidedMessage = "A service client must be provided to find a resource ID by name"

	CE_NoEndPointInCatalogCode    = "Com.1004" //client error
	CE_NoEndPointInCatalogMessage = "No suitable endpoint could be found in the service catalog."

	//Com1005
	CE_ApiNotFoundCode    = "Com.1005" //server error
	CE_ApiNotFoundMessage = "API not found"

	//1006
	CE_TimeoutErrorCode    = "Com.1006" //client error
	CE_TimeoutErrorMessage = "The request timed out %s times(%s for retry), perhaps we should have the threshold raised a little?"

	CE_ReauthExceedCode    = "Com.1006" //client error
	CE_ReauthExceedMessage = "Tried to re-authenticate 3 times with no success."

	CE_ReauthFuncErrorCode    = "Com.1006" //client error
	CE_ReauthFuncErrorMessage = "Get reauth function error [%s]"

	//Com2000
	//其他非典型错误，不再统一定义。
)

//UnifiedError, Unified definition of backend errors.
type UnifiedError struct {
	ErrCode    interface{} `json:"code"`
	ErrMessage string      `json:"message"`
}

//Initialize SDK client error.
func NewSystemCommonError(code, message string) error {
	return &UnifiedError{
		ErrCode:    code,
		ErrMessage: message,
	}
}

//NewSystemServerError,Handle background API error codes.
func NewSystemServerError(httpStatus int, responseContent string) error {
	//e.Body {"error": {"message": "instance is not shutoff.","code": "IMG.0008"}}
	return ParseSeverError(httpStatus, responseContent)
}

//Error,Implement the Error() interface.
func (e UnifiedError) Error() string {
	return fmt.Sprintf("{\"ErrorCode\":\"%s\",\"Message\":\"%s\"}", e.ErrCode, e.ErrMessage)
}

//ErrorCode,Error code converted to string type.
func (e UnifiedError) ErrorCode() string {
	if s, ok := e.ErrCode.(string); ok {
		return s
	}

	if i, ok := e.ErrCode.(int); ok {
		return string(i)
	}

	return ""
}

//Message,Return error message.
func (e UnifiedError) Message() string {
	return e.ErrMessage
}

// OneLevelError,Define the error code structure and match the error code of one layer of json structure
type OneLevelError struct {
	Message    string
	Request_id string
	ErrCode    string `json:"error_code"`
	ErrMsg     string `json:"error_msg"`
	Code	   string `json:"code"`
}

// ParseSeverError,This function uses json serialization to parse background API error codes.
func ParseSeverError(httpStatus int, responseContent string) error {
	//一层结构如下：
	//第一种：{"error_msg": "Instance *89973356-f733-418b-95b2-f6fc27244f18 could not be found.","err_code": 404}
	//第二种：{"message": "Instance *89973356-f733-418b-95b2-f6fc27244f18 could not be found.","code": "VPC.0101"}
	//第三种：html 页面，返回字符串形式，走正则匹配错误码。

	//两层结构如下：
	//{"itemNotFound": {"message": "Instance *89973356-f733-418b-95b2-f6fc27244f18 could not be found.", "code": 404}}
	//{"error": {"message": "instance is not shutoff.","code": "IMG.0008"}}
	var olErr OneLevelError
	var errMsg = make(map[string]UnifiedError)
	var isDevApi bool //是否为自研api接口
	var errCode string
	message := responseContent

	err := json.Unmarshal([]byte(responseContent), &errMsg)
	if err != nil { //一层结构错误
		err1 := json.Unmarshal([]byte(responseContent), &olErr)
		if err1 != nil {
			errCode = MatchErrorCode(httpStatus, message)
			message = responseContent
		} else {
			if olErr.Code == "" && olErr.ErrCode == "" {
				errCode = MatchErrorCode(httpStatus, olErr.Message)
				message = olErr.Message
			} else {
				if olErr.Code != "" {
					errCode = olErr.Code
					message = olErr.Message
				}
				if olErr.ErrCode != "" {
					errCode = olErr.ErrCode
					message = olErr.ErrMsg
				}
			}
		}
	} else { //两层结构错误
		for _, em := range errMsg {
			message = em.ErrMessage

			/*
				自研api的code字段为string且包含'.'，否则为原生api，
				原生api也可能是string类型，但是不包含'.'
				原生api的errCode走解析流程这里不需要赋值
				自研api的code示例:"IMG.0144"
				原生api的code示例:400或者"400"
			*/
			switch em.ErrCode.(type) {
			case string:
				if strings.Contains(em.ErrCode.(string), ".") {
					isDevApi = true
					errCode = em.ErrCode.(string)
				} else {
					isDevApi = false
				}
			default:
				isDevApi = false
			}
		}

		//原生api接口走解析流程
		if !isDevApi {
			errCode = MatchErrorCode(httpStatus, message)
		}
	}

	return &UnifiedError{
		ErrCode:    errCode,
		ErrMessage: message,
	}
}

//MatchErrorCode,Match the error code according to the error message
func MatchErrorCode(httpStatus int, message string) string {
	//common error
	if ok, _ := regexp.MatchString(CE_ApiNotFoundMessage, message); ok {
		return CE_ApiNotFoundCode
	}
	if ok, _ := regexp.MatchString(CE_StreamControlApiMessage, message); ok {
		return CE_StreamControlApiCode
	}

	//ECS error
	if ok, _ := regexp.MatchString(EcsAuthRequired, message); ok {
		return "Ecs.1499"
	}
	if ok, _ := regexp.MatchString(EcsPoilcyNotAllow, message); ok {
		return "Ecs.1500"
	}
	if ok, _ := regexp.MatchString(EcsTokenRoleEmpty, message); ok {
		return "Ecs.1500"
	}
	if ok, _ := regexp.MatchString(EcsTokenRoleForbidden, message); ok {
		return "Ecs.1500"
	}
	if ok, _ := regexp.MatchString(EcsErrorRoleToPerform, message); ok {
		return "Ecs.1500"
	}
	if ok, _ := regexp.MatchString(EcsQuotaExceeded, message); ok {
		return "Ecs.1501"
	}
	if ok, _ := regexp.MatchString(EcsPortNumberExceed, message); ok {
		return "Ecs.1502"
	}
	if ok, _ := regexp.MatchString(EcsVolumeNumberOver, message); ok {
		return "Ecs.1503"
	}
	if ok, _ := regexp.MatchString(EcsBlockImageNotFound, message); ok {
		return "Ecs.1511"
	}
	if ok, _ := regexp.MatchString(EcsImageNotFound, message); ok {
		return "Ecs.1511"
	}
	if ok, _ := regexp.MatchString(EcsFlavorNotFound, message); ok {
		return "Ecs.1512"
	}
	if ok, _ := regexp.MatchString(EcsInvalidFlavorRef, message); ok {
		return "Ecs.1512"
	}
	if ok, _ := regexp.MatchString(EcsNetworkNotFound, message); ok {
		return "Ecs.1513"
	}
	if ok, _ := regexp.MatchString(EcsBlockDevInvalid, message); ok {
		return "Ecs.1514"
	}
	if ok, _ := regexp.MatchString(EcsAZUnavailable, message); ok {
		return "Ecs.1515"
	}
	if ok, _ := regexp.MatchString(EcsSecurityGroupNotFound, message); ok {
		return "Ecs.1516"
	}
	if ok, _ := regexp.MatchString(EcsKeyPairNotFound, message); ok {
		return "Ecs.1517"
	}
	if ok, _ := regexp.MatchString(EcsInvalidKeyName, message); ok {
		return "Ecs.1517"
	}
	if ok, _ := regexp.MatchString(EcsInstanceGroupNotFound, message); ok {
		return "Ecs.1518"
	}
	if ok, _ := regexp.MatchString(EcsInvalidMetadata, message); ok {
		return "Ecs.1519"
	}
	if ok, _ := regexp.MatchString(EcsInvalidInputField, message); ok {
		return "Ecs.1519"
	}
	if ok, _ := regexp.MatchString(EcsUserDataBase64, message); ok {
		return "Ecs.1520"
	}
	if ok, _ := regexp.MatchString(EcsUserDataTooLarge, message); ok {
		return "Ecs.1521"
	}
	if ok, _ := regexp.MatchString(EcsInstanceDiskExceed, message); ok {
		return "Ecs.1522"
	}
	if ok, _ := regexp.MatchString(EcsFlavorMemoryNotEnough, message); ok {
		return "Ecs.1523"
	}
	if ok, _ := regexp.MatchString(EcsResourceSoldOut, message); ok {
		return "Ecs.1524"
	}
	if ok, _ := regexp.MatchString(EcsInstanceNotFound, message); ok {
		return "Ecs.1544"
	}
	if ok, _ := regexp.MatchString(EcsInstanceIsLocked, message); ok {
		return "Ecs.1545"
	}
	if ok, _ := regexp.MatchString(EcsInstCantBeOperated, message); ok {
		return "Ecs.1546"
	}
	if ok, _ := regexp.MatchString(EcsServerCantComply, message); ok {
		return "Ecs.1599"
	}
	if ok, _ := regexp.MatchString(EcsUnexpectedApiERROR, message); ok {
		return "Ecs.1599"
	}

	//IMS error
	if ok, _ := regexp.MatchString(Ims0027NoImageFoundWithId, message); ok {
		return "IMG.0027"
	}
	if ok, _ := regexp.MatchString(Ims0144FailedFindImage, message); ok {
		return "IMG.0144"
	}

	//ELB error
	if ok, _ := regexp.MatchString(ELB2001AdminStateUpFalse, message); ok {
		return "ELB.2001"
	}
	if ok, _ := regexp.MatchString(ELB2002IpNotValid, message); ok {
		return "ELB.2002"
	}
	if ok, _ := regexp.MatchString(ELB2003PoolNotFound, message); ok {
		return "ELB.2003"
	}
	if ok, _ := regexp.MatchString(ELB2004MemberNotSupportPort, message); ok {
		return "ELB.2004"
	}
	if ok, _ := regexp.MatchString(ELB2005SubnetMismatch, message); ok {
		return "ELB.2005"
	}
	if ok, _ := regexp.MatchString(ELB2006IpPortAlreadyPresent, message); ok {
		return "ELB.2006"
	}
	if ok, _ := regexp.MatchString(ELB2007MemberNotFound, message); ok {
		return "ELB.2007"
	}

	if ok, _ := regexp.MatchString(ELB6101QuotaExceeded, message); ok {
		return "ELB.6101"
	}
	if ok, _ := regexp.MatchString(ELB2541QuotaExceeded, message); ok {
		return "ELB.2541"
	}
	if ok, _ := regexp.MatchString(ELBb015QuotaExceeded, message); ok {
		return "ELB.B015"
	}
	if ok, _ := regexp.MatchString(ELB1071QuotaExceeded, message); ok {
		return "ELB.1071"
	}

	//没匹配上，用http状态码做error code
	return "Com." + strconv.Itoa(httpStatus)
}

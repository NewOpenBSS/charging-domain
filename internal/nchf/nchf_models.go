package nchf

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/fiorix/go-diameter/v4/diam"
)

type LocalDateTime time.Time

const localDateTimeLayout = "2006-01-02T15:04:05"

// UnmarshalJSON accepts either a JSON string in format yyyy-MM-dd'T'HH:mm:ss (no timezone)
// or an integer array [YYYY,MM,DD,hh,mm,ss] as returned by some implementations.
// The value is interpreted in time.Local unless you convert it after parsing.
func (t *LocalDateTime) UnmarshalJSON(b []byte) error {
	// Handle null
	if string(b) == "null" {
		*t = LocalDateTime(time.Time{})
		return nil
	}

	// Some CHF implementations encode date-time as an array: [YYYY,MM,DD,hh,mm,ss]
	// (optionally with more fields). Accept that too.
	if len(b) > 0 && b[0] == '[' {
		var parts []int
		if err := json.Unmarshal(b, &parts); err != nil {
			return err
		}
		if len(parts) < 6 {
			return fmt.Errorf("invalid LocalDateTime array (need at least 6 items): %v", parts)
		}
		tm := time.Date(parts[0], time.Month(parts[1]), parts[2], parts[3], parts[4], parts[5], 0, time.Local)
		*t = LocalDateTime(tm)
		return nil
	}

	// Default / spec-friendly encoding is a JSON string.
	s, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	if s == "" {
		*t = LocalDateTime(time.Time{})
		return nil
	}

	// First try the local no-timezone layout used by this project.
	tm, err := time.ParseInLocation(localDateTimeLayout, s, time.Local)
	if err == nil {
		*t = LocalDateTime(tm)
		return nil
	}

	// Be tolerant to RFC3339 timestamps (with timezone) if returned.
	if tm2, err2 := time.Parse(time.RFC3339Nano, s); err2 == nil {
		*t = LocalDateTime(tm2.In(time.Local))
		return nil
	}

	return err
}

// MarshalJSON outputs the time in format yyyy-MM-dd'T'HH:mm:ss (no timezone).
func (t LocalDateTime) MarshalJSON() ([]byte, error) {
	tm := time.Time(t)
	if tm.IsZero() {
		return []byte("null"), nil
	}
	return []byte(strconv.Quote(tm.Format(localDateTimeLayout))), nil
}

func (t LocalDateTime) Time() time.Time { return time.Time(t) }

type FinalUnitAction string

const (
	FinalUnitActionTerminate      FinalUnitAction = "TERMINATE"
	FinalUnitActionRedirect       FinalUnitAction = "REDIRECT"
	FinalUnitActionRestrictAccess FinalUnitAction = "RESTRICT_ACCESS"
)

func (r FinalUnitAction) Ordinal() int {
	switch r {
	default:
		fallthrough
	case FinalUnitActionTerminate:
		return 0
	case FinalUnitActionRedirect:
		return 1
	case FinalUnitActionRestrictAccess:
		return 2
	}
}

type NodeFunctionality string

const (
	NodeFunctionalityChf    NodeFunctionality = "CHF"
	NodeFunctionalityPcf    NodeFunctionality = "PCF"
	NodeFunctionalityNwdaf  NodeFunctionality = "NWDAF"
	NodeFunctionalityNef    NodeFunctionality = "NEF"
	NodeFunctionalitySmf    NodeFunctionality = "SMF"
	NodeFunctionalityAmf    NodeFunctionality = "AMF"
	NodeFunctionalitySmsf   NodeFunctionality = "SMSF"
	NodeFunctionalityUdm    NodeFunctionality = "UDM"
	NodeFunctionalityAusf   NodeFunctionality = "AUSF"
	NodeFunctionalityNssf   NodeFunctionality = "NSSF"
	NodeFunctionalityBsf    NodeFunctionality = "BSF"
	NodeFunctionalityGmlc   NodeFunctionality = "GMLC"
	NodeFunctionalityMme    NodeFunctionality = "MME"
	NodeFunctionalitySgsn   NodeFunctionality = "SGSN"
	NodeFunctionalityScsas  NodeFunctionality = "SCSAS"
	NodeFunctionalityMbSmf  NodeFunctionality = "MB_SMF"
	NodeFunctionalityMbUpf  NodeFunctionality = "MB_UPF"
	NodeFunctionalityGbaBsf NodeFunctionality = "GBA_BSF"
	NodeFunctionalityImsAs  NodeFunctionality = "IMS_AS"
	NodeFunctionalityCscf   NodeFunctionality = "CSCF"
	NodeFunctionalityHss    NodeFunctionality = "HSS"
	NodeFunctionalityAaa    NodeFunctionality = "AAA"
	NodeFunctionalityEir    NodeFunctionality = "EIR"
)

type RedirectAddressType string

const ()

type ResultCode string

const (
	ResultCodeSuccess                      ResultCode = "SUCCESS"
	ResultCodeEndUserServiceDenied         ResultCode = "END_USER_SERVICE_DENIED"
	ResultCodeQuotaManagementNotApplicable ResultCode = "QUOTA_MANAGEMENT_NOT_APPLICABLE"
	ResultCodeQuotaLimitReached            ResultCode = "QUOTA_LIMIT_REACHED"
	ResultCodeEndUserServiceRejected       ResultCode = "END_USER_SERVICE_REJECTED"
	ResultCodeUserUnknown                  ResultCode = "USER_UNKNOWN"
	ResultCodeRatingFailed                 ResultCode = "RATING_FAILED"
	ResultCodeQuotaManagement              ResultCode = "QUOTA_MANAGEMENT"
)

func (r ResultCode) DiamResultCode() uint32 {

	switch r {
	case ResultCodeSuccess:
		return diam.Success // DIAMETER_SUCCESS
	case ResultCodeEndUserServiceDenied:
		return 4010
	case ResultCodeQuotaManagementNotApplicable:
		return 4011
	case ResultCodeQuotaLimitReached:
		return 4012
	case ResultCodeEndUserServiceRejected:
		return 4013
	case ResultCodeUserUnknown:
		return 5030
	case ResultCodeRatingFailed:
		return 5031
	case ResultCodeQuotaManagement:
		return 5032
	default:
		return diam.UnableToComply
	}
}

func ResultCodePtr(v ResultCode) *ResultCode {
	r := v
	return &r
}

type RoleOfIMSNode string

const (
	RoleOfIMSNodeMo RoleOfIMSNode = "MO"
	RoleOfIMSNodeMt RoleOfIMSNode = "MT"
	RoleOfIMSNodeMf RoleOfIMSNode = "MF"
)

type TriggerCategory string

const ()

type TriggerType string

const ()

type AllocatedUnit struct {
	QuotaManagementIndicator *string                   `json:"quotaManagementIndicator,omitempty"`
	Triggers                 []Trigger                 `json:"triggers,omitempty"`
	TriggerTimestamp         *string                   `json:"triggerTimestamp,omitempty"`
	LocalSequenceNumber      *int                      `json:"localSequenceNumber,omitempty"`
	NsacContainerInformation *NSACContainerInformation `json:"nSACContainerInformation,omitempty"`
}

type CalledIdentityChange struct {
	ChangeTime        *string `json:"changeTime,omitempty"`
	OldCalledIdentity *string `json:"oldCalledIdentity,omitempty"`
	NewCalledIdentity *string `json:"newCalledIdentity,omitempty"`
}

type ChargingDataRequest struct {
	SubscriberIdentifier          *string                        `json:"subscriberIdentifier,omitempty"`
	ChargingId                    *string                        `json:"chargingId,omitempty"`
	InvocationSequenceNumber      *int64                         `json:"invocationSequenceNumber,omitempty"`
	RetransmissionIndicator       *bool                          `json:"retransmissionIndicator,omitempty"`
	InvocationTimeStamp           *LocalDateTime                 `json:"invocationTimeStamp,omitempty"`
	OneTimeEvent                  *bool                          `json:"oneTimeEvent,omitempty"`
	OneTimeEventType              *string                        `json:"oneTimeEventType,omitempty"`
	ServiceSpecificationInfo      *string                        `json:"serviceSpecificationInfo,omitempty"`
	EasId                         *string                        `json:"easId,omitempty"`
	EdnId                         *string                        `json:"ednId,omitempty"`
	EasProviderIdentifier         *string                        `json:"easProviderIdentifier,omitempty"`
	AmfId                         *string                        `json:"amfId,omitempty"`
	MultipleUnitUsage             []MultipleUnitUsage            `json:"multipleUnitUsage,omitempty"`
	Triggers                      []Trigger                      `json:"triggers,omitempty"`
	NfConsumerIdentification      *NFIdentification              `json:"nfConsumerIdentification,omitempty"`
	PduSessionChargingInformation *PDUSessionChargingInformation `json:"pduSessionChargingInformation,omitempty"`
	ImsChargingInformation        *IMSChargingInformation        `json:"imsChargingInformation,omitempty"`
	SmsChargingInformation        *SMSChargingInformation        `json:"smsChargingInformation,omitempty"`
	NefChargingInformation        *NEFChargingInformation        `json:"nefChargingInformation,omitempty"`

	requestType int
}

func NewChargingDataRequest() *ChargingDataRequest {
	req := &ChargingDataRequest{
		requestType:       0,
		MultipleUnitUsage: []MultipleUnitUsage{},
		Triggers:          []Trigger{},
	}
	return req
}
func (o *ChargingDataRequest) SetRequestType(requestType int) {
	o.requestType = requestType
}

func (o *ChargingDataRequest) GetRequestType() int {
	return o.requestType
}

func (o *ChargingDataRequest) ToJSON() ([]byte, error) {
	body, err := json.Marshal(o)

	if err != nil {
		return nil, err
	}

	return body, nil
}

func (o *ChargingDataRequest) GetChargeInformation() (string, any) {
	if o.PduSessionChargingInformation != nil {
		return "PDU", o.PduSessionChargingInformation
	} else if o.ImsChargingInformation != nil {
		return "IMS", o.ImsChargingInformation
	} else if o.NefChargingInformation != nil {
		return "NEF", o.NefChargingInformation
	} else if o.SmsChargingInformation != nil {
		return "SMS", o.SmsChargingInformation
	} else {
		return "", nil
	}
}

type LocationReportingChargingInformation struct {
	LocationEstimate *string `json:"locationEstimate,omitempty"`
}

type ChargingDataResponse struct {
	InvocationTimeStamp                  *LocalDateTime                        `json:"invocationTimeStamp,omitempty"`
	InvocationSequenceNumber             *int64                                `json:"invocationSequenceNumber,omitempty"`
	InvocationResult                     *InvocationResult                     `json:"invocationResult,omitempty"`
	SessionFailover                      *string                               `json:"sessionFailover,omitempty"`
	SupportedFeatures                    *string                               `json:"supportedFeatures,omitempty"`
	MultipleUnitInformation              []MultipleUnitInformation             `json:"multipleUnitInformation,omitempty"`
	Triggers                             []Trigger                             `json:"triggers,omitempty"`
	PDUSessionChargingInformation        *PDUSessionChargingInformation        `json:"pDUSessionChargingInformation,omitempty"`
	LocationReportingChargingInformation *LocationReportingChargingInformation `json:"locationReportingChargingInformation,omitempty"`
	Runtime                              *int64                                `json:"runtime,omitempty-"`
	requestType                          int
}

func NewChargingDataResponse() *ChargingDataResponse {
	resp := &ChargingDataResponse{requestType: 0}
	return resp
}
func (o *ChargingDataResponse) SetRequestType(requestType int) {
	o.requestType = requestType
}

func (o *ChargingDataResponse) GetRequestType() int {
	return o.requestType
}

func (o *ChargingDataResponse) GetRequestTypeString() string {
	switch o.requestType {
	case 1:
		return "INIT"
	case 2:
		return "UPDATE"
	case 3:
		return "TERMINATE"
	default:
		return "EVENT"
	}
}

type FinalUnitIndication struct {
	FinalUnitAction           *FinalUnitAction `json:"finalUnitAction,omitempty"`
	RestrictionFilterRule     *string          `json:"restrictionFilterRule,omitempty"`
	RestrictionFilterRuleList []string         `json:"restrictionFilterRuleList,omitempty"`
	FilterId                  *string          `json:"filterId,omitempty"`
	FilterIdList              []string         `json:"filterIdList,omitempty"`
	RedirectServer            *RedirectServer  `json:"redirectServer,omitempty"`
}

type GrantedUnit struct {
	TariffTimeChange     *string `json:"tariffTimeChange,omitempty"`
	Time                 *int64  `json:"time,omitempty"`
	TotalVolume          *int64  `json:"totalVolume,omitempty"`
	UplinkVolume         *int64  `json:"uplinkVolume,omitempty"`
	DownlinkVolume       *int64  `json:"downlinkVolume,omitempty"`
	ServiceSpecificUnits *int64  `json:"serviceSpecificUnits,omitempty"`
}

type IMSChargingInformation struct {
	EventType                           *string                `json:"eventType,omitempty"`
	ImsNodeFunctionality                *string                `json:"iMSNodeFunctionality,omitempty"`
	RoleOfNode                          *RoleOfIMSNode         `json:"roleOfNode,omitempty"`
	UserInformation                     *UserInformation       `json:"userInformation,omitempty"`
	UserLocationInfo                    *string                `json:"userLocationInfo,omitempty"`
	UeTimeZone                          *string                `json:"ueTimeZone,omitempty"`
	CallingPartyAddresses               []string               `json:"callingPartyAddresses,omitempty"`
	CalledPartyAddress                  *string                `json:"calledPartyAddress,omitempty"`
	ImsChargingIdentifier               *string                `json:"imsChargingIdentifier,omitempty"`
	PsData3gppOffStatus                 *string                `json:"3gppPSDataOffStatus,omitempty"`
	IsupCause                           *string                `json:"isupCause,omitempty"`
	ControlPlaneAddress                 *string                `json:"controlPlaneAddress,omitempty"`
	VlrNumber                           *string                `json:"vlrNumber,omitempty"`
	MscAddress                          *string                `json:"mscAddress,omitempty"`
	UserSessionID                       *string                `json:"userSessionID,omitempty"`
	OutgoingSessionID                   *string                `json:"outgoingSessionID,omitempty"`
	SessionPriority                     *string                `json:"sessionPriority,omitempty"`
	NumberPortabilityRoutingInformation *string                `json:"numberPortabilityRoutingInformation,omitempty"`
	CarrierSelectRoutingInformation     *string                `json:"carrierSelectRoutingInformation,omitempty"`
	AlternateChargedPartyAddress        *string                `json:"alternateChargedPartyAddress,omitempty"`
	RequestedPartyAddress               []string               `json:"requestedPartyAddress,omitempty"`
	CalledAssertedIdentities            []string               `json:"calledAssertedIdentities,omitempty"`
	CalledIdentityChanges               []CalledIdentityChange `json:"calledIdentityChanges,omitempty"`
	AssociatedURI                       []string               `json:"associatedURI,omitempty"`
}

type InvocationResult struct {
	Error           *ProblemDetails `json:"error,omitempty"`
	FailureHandling *string         `json:"failureHandling,omitempty"`
}

type MultipleUnitInformation struct {
	ResultCode           *ResultCode          `json:"resultCode,omitempty"`
	RatingGroup          *int64               `json:"ratingGroup,omitempty"`
	GrantedUnit          *GrantedUnit         `json:"grantedUnit,omitempty"`
	AllocatedUnit        *AllocatedUnit       `json:"allocatedUnit,omitempty"`
	Triggers             []Trigger            `json:"triggers,omitempty"`
	ValidityTime         *int32               `json:"validityTime,omitempty"`
	QuotaHoldingTime     *int                 `json:"quotaHoldingTime,omitempty"`
	FinalUnitIndication  *FinalUnitIndication `json:"finalUnitIndication,omitempty"`
	TimeQuotaThreshold   *int                 `json:"timeQuotaThreshold,omitempty"`
	VolumeQuotaThreshold *int64               `json:"volumeQuotaThreshold,omitempty"`
	UnitQuotaThreshold   *int                 `json:"unitQuotaThreshold,omitempty"`
	UpfId                *string              `json:"uPFID,omitempty"`
	MbupfId              *string              `json:"mBUPFID,omitempty"`
	QosProfile           *string              `json:"qoSProfile,omitempty"`
}

type MultipleUnitUsage struct {
	RatingGroup          *int64              `json:"ratingGroup,omitempty"`
	RequestedUnit        *RequestedUnit      `json:"requestedUnit,omitempty"`
	UsedUnitContainer    []UsedUnitContainer `json:"usedUnitContainer,omitempty"`
	AllocatedUnit        *AllocatedUnit      `json:"allocatedUnit,omitempty"`
	UpfID                *string             `json:"uPFID,omitempty"`
	MultihomedPDUAddress *PDUAddress         `json:"multihomedPDUAddress,omitempty"`
	MbupfID              *string             `json:"mBUPFID,omitempty"`
}

type NEFChargingInformation struct {
	ExternalIndividualIdentifier *string           `json:"externalIndividualIdentifier,omitempty"`
	ExternalIndividualIdList     []string          `json:"externalIndividualIdList,omitempty"`
	InternalIndividualIdentifier *string           `json:"internalIndividualIdentifier,omitempty"`
	InternalIndividualIdList     []string          `json:"internalIndividualIdList,omitempty"`
	ExternalGroupIdentifier      *string           `json:"externalGroupIdentifier,omitempty"`
	GroupIdentifier              *string           `json:"groupIdentifier,omitempty"`
	ApiDirection                 *string           `json:"aPIDirection,omitempty"`
	ApiTargetNetworkFunction     *NFIdentification `json:"aPITargetNetworkFunction,omitempty"`
	ApiResultCode                *int              `json:"aPIResultCode,omitempty"`
	ApiName                      *string           `json:"aPIName,omitempty"`
	ApiReference                 *string           `json:"aPIReference,omitempty"`
	ApiOperation                 *string           `json:"aPIOperation,omitempty"`
	ApiContent                   *string           `json:"aPIContent,omitempty"`
}

type NFIdentification struct {
	NfName            *string            `json:"nFName,omitempty"`
	NfIPv4Address     *string            `json:"nFIPv4Address,omitempty"`
	NfIPv6Address     *string            `json:"nFIPv6Address,omitempty"`
	NfPLMNID          *PlmnId            `json:"nFPLMNID,omitempty"`
	NodeFunctionality *NodeFunctionality `json:"nodeFunctionality,omitempty"`
	NfFqdn            *string            `json:"nFFqdn,omitempty"`
}

type NSACContainerInformation struct {
	NumberOfUEs  *int `json:"numberOfUEs,omitempty"`
	NumberOfPDUs *int `json:"numberOfPDUs,omitempty"`
}

type OriginatorInfo struct {
	OriginatorSUPI            *string        `json:"originatorSUPI,omitempty"`
	OriginatorGPSI            *string        `json:"originatorGPSI,omitempty"`
	OriginatorOtherAddress    *SMAddressInfo `json:"originatorOtherAddress,omitempty"`
	OriginatorReceivedAddress *SMAddressInfo `json:"originatorReceivedAddress,omitempty"`
	OriginatorSCCPAddress     *string        `json:"originatorSCCPAddress,omitempty"`
	SmOriginatorInterface     *SMInterface   `json:"sMOriginatorInterface,omitempty"`
	SmOriginatorProtocolId    *string        `json:"sMOriginatorProtocolId,omitempty"`
}

type PDUAddress struct {
	PduIPv4Address           *string  `json:"pduIPv4Address,omitempty"`
	PduIPv6AddresswithPrefix *string  `json:"pduIPv6AddresswithPrefix,omitempty"`
	PduAddressPrefixLength   *int     `json:"pduAddressprefixlength,omitempty"`
	IPv4dynamicAddressFlag   *bool    `json:"iPv4dynamicAddressFlag,omitempty"`
	IPv6dynamicPrefixFlag    *bool    `json:"iPv6dynamicPrefixFlag,omitempty"`
	AddIpv6AddrPrefixes      *string  `json:"addIpv6AddrPrefixes,omitempty"`
	AddIpv6AddrPrefixList    []string `json:"addIpv6AddrPrefixList,omitempty"`
}

type PDUContainerInformation struct {
	TimeofFirstUsage                   *string `json:"timeofFirstUsage,omitempty"`
	TimeofLastUsage                    *string `json:"timeofLastUsage,omitempty"`
	QosInformation                     *string `json:"qoSInformation,omitempty"`
	QosCharacteristics                 *string `json:"qoSCharacteristics,omitempty"`
	AfChargingIdentifier               *string `json:"afChargingIdentifier,omitempty"`
	AfChargingIdString                 *string `json:"afChargingIdString,omitempty"`
	UserLocationInformation            *string `json:"userLocationInformation,omitempty"`
	UeTimeZone                         *string `json:"uetimeZone,omitempty"`
	RatType                            *string `json:"rATType,omitempty"`
	SponsorIdentity                    *string `json:"sponsorIdentity,omitempty"`
	ApplicationServiceProviderIdentity *string `json:"applicationserviceProviderIdentity,omitempty"`
	ChargingRuleBaseName               *string `json:"chargingRuleBaseName,omitempty"`
}

type PDUSessionChargingInformation struct {
	ChargingId                   *string          `json:"chargingId,omitempty"`
	SmfChargingid                *string          `json:"sMFchargingId,omitempty"`
	HomeProvidedChargingId       *string          `json:"homeProvidedChargingId,omitempty"`
	SmfHomeProvidedChargingId    *string          `json:"sMFHomeProvidedChargingId,omitempty"`
	UserInformation              *UserInformation `json:"userInformation,omitempty"`
	UserLocationInfo             *string          `json:"userLocationinfo,omitempty"`
	ImsSessionInformation        *string          `json:"iMSSessionInformation,omitempty"`
	MapduNon3GPPUserLocationInfo *string          `json:"mAPDUNon3GPPUserLocationInfo,omitempty"`
	Non3GPPUserLocationTime      *string          `json:"non3GPPUserLocationTime,omitempty"`
	MapduNon3GPPUserLocationTime *string          `json:"mAPDUNon3GPPUserLocationTime,omitempty"`
	UeTimeZone                   *string          `json:"uetimeZone,omitempty"`
	PduSessionInformation        *string          `json:"pduSessionInformation,omitempty"`
	UnitCountInactivityTimer     *string          `json:"unitCountInactivityTimer,omitempty"`
	RanSecondaryRATUsageReport   *string          `json:"rANSecondaryRATUsageReport,omitempty"`
}

type PlmnId struct {
	Mcc *string `json:"mcc,omitempty"`
	Mnc *string `json:"mnc,omitempty"`
}

type RecipientInfo struct {
	RecipientSUPI            *string        `json:"recipientSUPI,omitempty"`
	RecipientGPSI            *string        `json:"recipientGPSI,omitempty"`
	RecipientOtherAddress    *SMAddressInfo `json:"recipientOtherAddress,omitempty"`
	RecipientReceivedAddress *SMAddressInfo `json:"recipientReceivedAddress,omitempty"`
	RecipientSCCPAddress     *string        `json:"recipientSCCPAddress,omitempty"`
	SmDestinationInterface   *SMInterface   `json:"sMDestinationInterface,omitempty"`
	SmRecipientProtocolId    *string        `json:"sMrecipientProtocolId,omitempty"`
}

type RedirectServer struct {
	RedirectAddressType   *RedirectAddressType `json:"redirectAddressType,omitempty"`
	RedirectServerAddress *string              `json:"redirectServerAddress,omitempty"`
}

type RequestedUnit struct {
	Time                 *int   `json:"time,omitempty"`
	TotalVolume          *int64 `json:"totalVolume,omitempty"`
	UplinkVolume         *int64 `json:"uplinkVolume,omitempty"`
	DownlinkVolume       *int64 `json:"downlinkVolume,omitempty"`
	ServiceSpecificUnits *int64 `json:"serviceSpecificUnits,omitempty"`
}

type SMAddressDomain struct {
	DomainName *string `json:"domainName,omitempty"`
	ImsiMccMnc *string `json:"3GPPIMSIMCCMNC,omitempty"`
}

type SMAddressInfo struct {
	SmAddressType   *string          `json:"sMaddressType,omitempty"`
	SmAddressData   *string          `json:"sMaddressData,omitempty"`
	SmAddressDomain *SMAddressDomain `json:"sMaddressDomain,omitempty"`
}

type SMInterface struct {
	InterfaceId   *string `json:"interfaceId,omitempty"`
	InterfaceText *string `json:"interfaceText,omitempty"`
	InterfacePort *string `json:"interfacePort,omitempty"`
	InterfaceType *string `json:"interfaceType,omitempty"`
}

type SMSChargingInformation struct {
	OriginatorInfo          *OriginatorInfo `json:"originatorInfo,omitempty"`
	RecipientInfo           []RecipientInfo `json:"recipientInfo,omitempty"`
	UserEquipmentInfo       *string         `json:"userEquipmentInfo,omitempty"`
	RoamerInOut             *string         `json:"roamerInOut,omitempty"`
	UserLocationInfo        *string         `json:"userLocationinfo,omitempty"`
	UeTimeZone              *string         `json:"uetimeZone,omitempty"`
	RatType                 *string         `json:"rATType,omitempty"`
	SmscAddress             *string         `json:"sMSCAddress,omitempty"`
	SmDataCodingScheme      *int            `json:"sMDataCodingScheme,omitempty"`
	SmMessageType           *string         `json:"sMMessageType,omitempty"`
	SmReplyPathRequested    *string         `json:"sMReplyPathRequested,omitempty"`
	SmUserDataHeader        *string         `json:"sMUserDataHeader,omitempty"`
	SmStatus                *string         `json:"sMStatus,omitempty"`
	SmDischargeTime         *string         `json:"sMDischargeTime,omitempty"`
	NumberOfMessagesSent    *int            `json:"numberofMessagesSent,omitempty"`
	SmServiceType           *string         `json:"sMServiceType,omitempty"`
	SmSequenceNumber        *int            `json:"sMSequenceNumber,omitempty"`
	SmSresult               *int            `json:"sMSresult,omitempty"`
	SubmissionTime          *string         `json:"submissionTime,omitempty"`
	SmPriority              *string         `json:"sMPriority,omitempty"`
	MessageReference        *string         `json:"messageReference,omitempty"`
	MessageSize             *int            `json:"messageSize,omitempty"`
	MessageClass            *string         `json:"messageClass,omitempty"`
	DeliveryReportRequested *string         `json:"deliveryReportRequested,omitempty"`
}

type Trigger struct {
	TriggerType      *TriggerType     `json:"triggerType,omitempty"`
	TriggerCategory  *TriggerCategory `json:"triggerCategory,omitempty"`
	TimeLimit        *int             `json:"timeLimit,omitempty"`
	VolumeLimit      *int             `json:"volumeLimit,omitempty"`
	VolumeLimit64    *int64           `json:"volumeLimit64,omitempty"`
	EventLimit       *int             `json:"eventLimit,omitempty"`
	MaxNumberOfccc   *int             `json:"maxNumberOfccc,omitempty"`
	TariffTimeChange *string          `json:"tariffTimeChange,omitempty"`
}

type UsedUnitContainer struct {
	ServiceId                *string                  `json:"serviceId,omitempty"`
	QuotaManagementIndicator *string                  `json:"quotaManagementIndicator,omitempty"`
	Reclaimable              *bool                    `json:"reclaimable,omitempty"`
	Triggers                 []Trigger                `json:"triggers,omitempty"`
	TriggerTimestamp         *string                  `json:"triggerTimestamp,omitempty"`
	Time                     *int                     `json:"time,omitempty"`
	TotalVolume              *int64                   `json:"totalVolume,omitempty"`
	UplinkVolume             *int64                   `json:"uplinkVolume,omitempty"`
	DownlinkVolume           *int64                   `json:"downlinkVolume,omitempty"`
	ServiceSpecificUnits     *int64                   `json:"serviceSpecificUnits,omitempty"`
	EventTimeStamps          []string                 `json:"eventTimeStamps,omitempty"`
	LocalSequenceNumber      *int                     `json:"localSequenceNumber,omitempty"`
	PduContainerInformation  *PDUContainerInformation `json:"pDUContainerInformation,omitempty"`
}

type UserInformation struct {
	ServedGPSI          *string `json:"servedGPSI,omitempty"`
	ServedPEI           *string `json:"servedPEI,omitempty"`
	UnauthenticatedFlag *bool   `json:"unauthenticatedFlag,omitempty"`
	RoamerInOut         *string `json:"roamerInOut,omitempty"`
}

type Cause string

const (
	CauseNoGrantsForUsage     Cause = "NO_GRANTS_FOR_USAGE"
	CauseMoreUsagesThanGrants Cause = "MORE_USAGES_THAN_GRANTS"
	CauseServiceBarred        Cause = "SERVICE_BARRED"
	CauseSubscriberInactive   Cause = "SUBSCRIBER_INACTIVE"
	CauseSubscriberNotFound   Cause = "SUBSCRIBER_NOT_FOUND"
	CauseUnableToClassify     Cause = "UNABLE_TO_CLASSIFY"
	CauseUnknownCarrier       Cause = "UNKNOWN_CARRIER"
	CauseUnknownDestination   Cause = "UNKNOWN_DESTINATION"
	CauseUnknownSubscriber    Cause = "UNKNOWN_SUBSCRIBER"
	CauseUsedMoreThanGranted  Cause = "USED_MORE_THAN_GRANTED"
	CauseUnknownNumberPlan    Cause = "UNKNOWN_NUMBER_PLAN"
	CauseNoRatingEntry        Cause = "NO_RATING_ENTRY"

	CauseDatabaseError      Cause = "DATABASE_ERROR"
	CauseInvalidReferenceID Cause = "INVALID_REFERENCE_ID"
	CauseRuleEvaluatorError Cause = "RULE_EVALUATOR_ERROR"
	CauseSystemError        Cause = "SYSTEM_ERROR"

	CauseInsufficientQuota Cause = "INSUFFICIENT_QUOTA"
)

type InvalidParam struct {
	Param  string `json:"param,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type ProblemDetails struct {
	Type              *string        `json:"type,omitempty"`
	Title             *string        `json:"title,omitempty"`
	Status            *int           `json:"status,omitempty"`
	Detail            *string        `json:"detail,omitempty"`
	Instance          *string        `json:"instance,omitempty"`
	Cause             *Cause         `json:"cause,omitempty"`
	InvalidParams     []InvalidParam `json:"invalidParams,omitempty"`
	SupportedFeatures *string        `json:"supportedFeatures,omitempty"`
}

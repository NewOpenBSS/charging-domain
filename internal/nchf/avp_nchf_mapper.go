// Package nchf provides mapping between Nchf (5G CHF) JSON models and Diameter DCCA (Gy/Ro) AVPs.
package nchf

import (
	"fmt"

	"go-ocs/internal/logging"

	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
)

const (
	SystemError               uint32 = 4000
	CreditControlLimitReached uint32 = 4012
)

const (
	AVP_USSD_INFORMATION uint32 = 885
	AVP_USSD_STRING      uint32 = 827
)

// NhcfToAvpResponse maps a ChargingDataResponse into a Diameter Answer message and returns the Diameter result code.
func NhcfToAvpResponse(answer *diam.Message, response *ChargingDataResponse) (uint32, error) {

	if _, err := answer.NewAVP(avp.CCRequestType, avp.Mbit, 0, datatype.Unsigned32(response.requestType)); err != nil {
		logging.Error("failed to add RequestType AVP", "error", err)
	}

	if response.InvocationResult != nil && response.InvocationResult.Error != nil {
		return mapErrorResponse(answer, response.InvocationResult.Error)
	}

	return mapChargingDataResponse(answer, response)
}

// mapErrorResponse applies ProblemDetails from an invocation error to a Diameter Answer and returns the mapped result code.
func mapErrorResponse(answer *diam.Message, problemDetails *ProblemDetails) (uint32, error) {

	if problemDetails != nil {
		code := diameterResultCodeFromCause(*problemDetails.Cause)
		if a, err := answer.FindAVP(avp.ResultCode, 0); err == nil {
			a.Data = datatype.Unsigned32(code)
		} else {
			answer.NewAVP(avp.ResultCode, avp.Mbit, 0, datatype.Unsigned32(code))
		}

		if problemDetails.Detail != nil {
			if _, err := answer.NewAVP(avp.ErrorMessage, avp.Mbit, 0, datatype.UTF8String(*problemDetails.Detail)); err != nil {
				logging.Error("failed to add ErrorMessage AVP", "error", err)
				return diam.UnableToComply, err
			}
		}

		return code, nil
	}

	return diam.UnableToComply, nil
}

// mapChargingDataResponse maps a successful ChargingDataResponse payload to Diameter Answer AVPs.
func mapChargingDataResponse(answer *diam.Message, response *ChargingDataResponse) (uint32, error) {

	// Map invocation timestamp
	if response.InvocationTimeStamp != nil {
		if _, err := answer.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(response.InvocationTimeStamp.Time())); err != nil {
			return diam.UnableToComply, fmt.Errorf("failed to add Event Timestamp AVP: %w", err)
		}
	}

	// Map invocation sequence number
	if response.InvocationSequenceNumber != nil {
		if _, err := answer.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(*response.InvocationSequenceNumber)); err != nil {
			return diam.UnableToComply, fmt.Errorf("failed to add CC Request Number AVP: %w", err)
		}
	}

	// Map multiple unit information
	if err := mapMultipleUnitInformation(answer, response.MultipleUnitInformation); err != nil {
		return diam.UnableToComply, err
	}

	return diam.Success, nil
}

// mapMultipleUnitInformation maps each MultipleUnitInformation item to an MSCC (Multiple-Services-Credit-Control) grouped AVP.
func mapMultipleUnitInformation(answer *diam.Message, multipleUnitInformationList []MultipleUnitInformation) error {
	if len(multipleUnitInformationList) == 0 {
		return nil
	}

	for i := range multipleUnitInformationList {
		mui := &multipleUnitInformationList[i]
		mscc := buildMSCC(mui)

		// Create a grouped AVP for each multiple unit information
		if _, err := answer.NewAVP(avp.MultipleServicesCreditControl, avp.Mbit, 0, mscc); err != nil {
			return fmt.Errorf("failed to add Multiple Services Credit Control Group: %w", err)
		}
	}

	return nil
}

// buildMSCC builds an MSCC (Multiple-Services-Credit-Control) grouped AVP from a MultipleUnitInformation item.
func buildMSCC(mui *MultipleUnitInformation) *diam.GroupedAVP {
	mscc := &diam.GroupedAVP{AVP: []*diam.AVP{}}

	appendMSCCRatingGroup(mscc, mui)
	appendMSCCResultCode(mscc, mui)
	appendMSCCGrantedUnit(mscc, mui)
	appendMSCCValidityTime(mscc, mui)
	appendMSCCQuotaHoldingTime(mscc, mui)
	appendMSCCFinalUnitIndication(mscc, mui)

	return mscc
}

// appendMSCCRatingGroup appends the Rating-Group AVP to the MSCC if present and non-zero.
func appendMSCCRatingGroup(mscc *diam.GroupedAVP, mui *MultipleUnitInformation) {
	// Map rating group. Zero is a special case indicating no rating group is specified.
	if mui.RatingGroup == nil || *mui.RatingGroup == 0 {
		return
	}
	mscc.AVP = append(mscc.AVP,
		diam.NewAVP(avp.RatingGroup, avp.Mbit, 0, datatype.Unsigned32(*mui.RatingGroup)),
	)
}

// appendMSCCResultCode appends the per-MSCC Result-Code AVP if present.
func appendMSCCResultCode(mscc *diam.GroupedAVP, mui *MultipleUnitInformation) {
	if mui.ResultCode == nil {
		return
	}
	mscc.AVP = append(mscc.AVP,
		diam.NewAVP(avp.ResultCode, avp.Mbit, 0, datatype.Unsigned32((*mui.ResultCode).DiamResultCode())),
	)
}

// appendMSCCGrantedUnit appends the Granted-Service-Unit grouped AVP if present.
func appendMSCCGrantedUnit(mscc *diam.GroupedAVP, mui *MultipleUnitInformation) {
	if mui.GrantedUnit == nil {
		return
	}

	gsu := buildGrantedServiceUnit(*mui.GrantedUnit)
	if gsu == nil {
		return
	}

	mscc.AVP = append(mscc.AVP,
		diam.NewAVP(avp.GrantedServiceUnit, avp.Mbit, 0, gsu),
	)
}

// buildGrantedServiceUnit builds a Granted-Service-Unit grouped AVP from a GrantedUnit.
// It returns nil if the granted unit contains no usable fields.
func buildGrantedServiceUnit(grantedUnit GrantedUnit) *diam.GroupedAVP {
	gsu := &diam.GroupedAVP{AVP: []*diam.AVP{}}

	if grantedUnit.Time != nil {
		gsu.AVP = append(gsu.AVP,
			diam.NewAVP(avp.CCTime, avp.Mbit, 0, datatype.Unsigned32(*grantedUnit.Time)),
		)
	}

	if grantedUnit.TotalVolume != nil {
		gsu.AVP = append(gsu.AVP,
			diam.NewAVP(avp.CCTotalOctets, avp.Mbit, 0, datatype.Unsigned64(*grantedUnit.TotalVolume)),
		)
	}

	if grantedUnit.UplinkVolume != nil {
		gsu.AVP = append(gsu.AVP,
			diam.NewAVP(avp.CCInputOctets, avp.Mbit, 0, datatype.Unsigned64(*grantedUnit.UplinkVolume)),
		)
	}

	if grantedUnit.DownlinkVolume != nil {
		gsu.AVP = append(gsu.AVP,
			diam.NewAVP(avp.CCOutputOctets, avp.Mbit, 0, datatype.Unsigned64(*grantedUnit.DownlinkVolume)),
		)
	}

	if grantedUnit.ServiceSpecificUnits != nil {
		gsu.AVP = append(gsu.AVP,
			diam.NewAVP(avp.CCServiceSpecificUnits, avp.Mbit, 0, datatype.Unsigned64(*grantedUnit.ServiceSpecificUnits)),
		)
	}

	if len(gsu.AVP) == 0 {
		return nil
	}

	return gsu
}

// appendMSCCValidityTime appends the Validity-Time AVP to the MSCC if present.
func appendMSCCValidityTime(mscc *diam.GroupedAVP, mui *MultipleUnitInformation) {
	if mui.ValidityTime == nil {
		return
	}
	mscc.AVP = append(mscc.AVP,
		diam.NewAVP(avp.ValidityTime, avp.Mbit, 0, datatype.Unsigned32(*mui.ValidityTime)),
	)
}

// appendMSCCQuotaHoldingTime appends the Quota-Holding-Time AVP to the MSCC if present.
func appendMSCCQuotaHoldingTime(mscc *diam.GroupedAVP, mui *MultipleUnitInformation) {
	if mui.QuotaHoldingTime == nil {
		return
	}
	mscc.AVP = append(mscc.AVP,
		diam.NewAVP(avp.QuotaHoldingTime, avp.Mbit, 0, datatype.Unsigned32(*mui.QuotaHoldingTime)),
	)
}

// appendMSCCFinalUnitIndication appends the Final-Unit-Indication grouped AVP to the MSCC if present.
func appendMSCCFinalUnitIndication(mscc *diam.GroupedAVP, mui *MultipleUnitInformation) {
	if mui.FinalUnitIndication == nil {
		return
	}

	fui := &diam.GroupedAVP{AVP: []*diam.AVP{}}
	fui.AVP = append(fui.AVP,
		diam.NewAVP(avp.FinalUnitAction, avp.Mbit, 0, datatype.Enumerated((*mui.FinalUnitIndication.FinalUnitAction).Ordinal())),
	)

	mscc.AVP = append(mscc.AVP,
		diam.NewAVP(avp.FinalUnitIndication, avp.Mbit, 0, fui),
	)
}

// diameterResultCodeFromCause maps an Nchf ProblemDetails cause to an appropriate Diameter Result-Code.
func diameterResultCodeFromCause(cause Cause) uint32 {
	switch cause {
	case CauseNoGrantsForUsage,
		CauseMoreUsagesThanGrants,
		CauseServiceBarred,
		CauseSubscriberInactive,
		CauseSubscriberNotFound,
		CauseUnableToClassify,
		CauseUnknownCarrier,
		CauseUnknownDestination,
		CauseUnknownSubscriber,
		CauseUsedMoreThanGranted,
		CauseUnknownNumberPlan,
		CauseNoRatingEntry:
		return diam.UnableToComply

	case CauseDatabaseError,
		CauseInvalidReferenceID,
		CauseRuleEvaluatorError,
		CauseSystemError:
		return SystemError // whatever constant you use

	case CauseInsufficientQuota:
		return CreditControlLimitReached

	default:
		// unknown cause: choose a safe default (5012 is usually fine)
		return diam.UnableToComply
	}
}

// AvpToNchfRequest builds a ChargingDataRequest from a Diameter CCR message by extracting relevant AVPs.
func AvpToNchfRequest(m *diam.Message) (*ChargingDataRequest, error) {

	request := NewChargingDataRequest()

	// Extract subscriber identifier (MSISDN)
	request.SubscriberIdentifier = extractSubscriberIdentifier(m)

	// Extract charging ID (Session ID)
	if sessId, ok := extractAvpString(m, avp.SessionID); ok {
		request.ChargingId = &sessId
	}

	// Extract invocation sequence number
	if val, ok := extractAvpInteger(m, avp.CCRequestNumber); ok {
		i64 := int64(val)
		request.InvocationSequenceNumber = &i64
	}

	// Extract request type
	if requestType, ok := extractAvpInteger(m, avp.CCRequestType); ok {
		oneTimeEventType := false
		if requestType == 4 {
			oneTimeEventType = true
		}
		request.OneTimeEvent = &oneTimeEventType
		request.SetRequestType(requestType)
	}

	// Extract timestamp
	if ts, ok := extractTimestamp(m, avp.EventTimestamp); ok {
		request.InvocationTimeStamp = &ts
	}

	// Extract service information
	if s, ok := extractAvpString(m, avp.ServiceIdentifier); ok {
		request.ServiceSpecificationInfo = &s
	}

	// Extract multiple unit usage information
	if units, ok := extractMultipleUnitUsage(m); ok {
		request.MultipleUnitUsage = units
	}

	// Extract NF identification
	if nfId, ok := extractNFIdentification(m); ok {
		request.NfConsumerIdentification = &nfId
	}

	if imsInfo, ok := extractIMSChargingInformation(m); ok {
		request.ImsChargingInformation = imsInfo
	}

	if smsInfo, ok := extractSMSChargingInformation(m); ok {
		request.SmsChargingInformation = smsInfo
	}

	if nefInfo, ok := extractNEFChargingInformation(m); ok {
		request.NefChargingInformation = nefInfo
	}

	if pduInfo, ok := extractPDUSessionChargingInformation(m); ok {
		request.PduSessionChargingInformation = pduInfo
	}

	return request, nil
}

// extractPDUSessionChargingInformation extracts PDU session charging information from Service-Information/PS-Information AVPs.
func extractPDUSessionChargingInformation(m *diam.Message) (*PDUSessionChargingInformation, bool) {

	var gAvp *diam.AVP
	for _, avp_item := range m.AVP {
		if avp_item.Code == avp.ServiceInformation {
			gAvp = avp_item
			break
		}
	}
	if gAvp == nil {
		return nil, false
	}

	g, ok := gAvp.Data.(*diam.GroupedAVP)
	if !ok {
		return nil, false
	}

	if _, ok := findInGroup(g, avp.PSInformation); !ok {
		return nil, false
	}

	// Extract charging ID
	chargingId, ok := extractAvpString(m, avp.SessionID)
	if !ok {
		return nil, false
	}

	pdu := PDUSessionChargingInformation{ChargingId: &chargingId}

	return &pdu, true
}

// extractNEFChargingInformation extracts NEF charging information (e.g., USSD identifiers) from Service-Information AVPs.
func extractNEFChargingInformation(m *diam.Message) (*NEFChargingInformation, bool) {

	var gAvp *diam.AVP
	for _, avp_item := range m.AVP {
		if avp_item.Code == avp.ServiceInformation {
			gAvp = avp_item
			break
		}
	}
	if gAvp == nil {
		return nil, false
	}

	g, ok := gAvp.Data.(*diam.GroupedAVP)
	if !ok {
		return nil, false
	}

	ussdInfo, ok := findInGroup(g, AVP_USSD_INFORMATION)
	if !ok {
		return nil, false
	}

	// Extract external individual identifier
	ussd, ok := findInGroup(ussdInfo.Data.(*diam.GroupedAVP), AVP_USSD_STRING)
	if !ok {
		return nil, false
	}

	ussdString := string(ussd.Data.(datatype.UTF8String))
	nef := NEFChargingInformation{ExternalGroupIdentifier: &ussdString}

	return &nef, true
}

// extractSMSChargingInformation extracts SMS charging information from Service-Information/SMS-Information AVPs.
func extractSMSChargingInformation(m *diam.Message) (*SMSChargingInformation, bool) {
	g, ok := findServiceInformationGroup(m)
	if !ok {
		return nil, false
	}

	sms, ok := findGroup(g, avp.SMSInformation)
	if !ok {
		return nil, false
	}

	smsInfo := &SMSChargingInformation{}

	appendSMSOriginatorInfo(smsInfo, sms)
	appendSMSMessageType(smsInfo, sms)
	appendSMSRecipientInfo(smsInfo, sms)
	appendSMSStatus(smsInfo, sms)
	appendSMSCAddress(smsInfo, sms)

	return smsInfo, true
}

// findServiceInformationGroup returns the Service-Information grouped AVP from a Diameter message.
func findServiceInformationGroup(m *diam.Message) (*diam.GroupedAVP, bool) {
	for _, a := range m.AVP {
		if a != nil && a.Code == avp.ServiceInformation {
			g, ok := a.Data.(*diam.GroupedAVP)
			if !ok || g == nil {
				return nil, false
			}
			return g, true
		}
	}
	return nil, false
}

// appendSMSOriginatorInfo populates OriginatorInfo fields from SMS-Information.
func appendSMSOriginatorInfo(out *SMSChargingInformation, sms *diam.GroupedAVP) {
	if a, ok := findInGroup(sms, avp.OriginatorInterface); ok {
		if s, ok := a.Data.(datatype.UTF8String); ok {
			originatorSUPI := string(s)
			i := OriginatorInfo{OriginatorSUPI: &originatorSUPI}
			out.OriginatorInfo = &i
		}
	}
}

// appendSMSMessageType populates the SMS message type from SMS-Information.
func appendSMSMessageType(out *SMSChargingInformation, sms *diam.GroupedAVP) {
	if a, ok := findInGroup(sms, avp.MessageType); ok {
		if s, ok := a.Data.(datatype.UTF8String); ok {
			smMessageType := string(s)
			out.SmMessageType = &smMessageType
		}
	}
}

// appendSMSRecipientInfo populates recipient addressing information from SMS-Information.
func appendSMSRecipientInfo(out *SMSChargingInformation, sms *diam.GroupedAVP) {
	avs, ok := findAvpsInGroup(sms, avp.RecipientInfo)
	if !ok {
		return
	}

	recipients := make([]RecipientInfo, 0, len(avs))
	for _, av := range avs {
		if av.Data == nil {
			continue
		}
		riGrp, ok := av.Data.(*diam.GroupedAVP)
		if !ok || riGrp == nil {
			continue
		}

		addrAvp, ok := findInGroup(riGrp, avp.RecipientAddress)
		if !ok || addrAvp == nil || addrAvp.Data == nil {
			continue
		}
		addrGrp, ok := addrAvp.Data.(*diam.GroupedAVP)
		if !ok || addrGrp == nil {
			continue
		}

		otherAddr := extractSMAddressInfo(addrGrp)
		recipients = append(recipients, RecipientInfo{RecipientOtherAddress: &otherAddr})
	}

	if len(recipients) > 0 {
		out.RecipientInfo = recipients
	}
}

// extractSMAddressInfo extracts SM address data and address type from a Recipient-Address grouped AVP.
func extractSMAddressInfo(addrGrp *diam.GroupedAVP) SMAddressInfo {
	otherAddr := SMAddressInfo{}

	// Address-Data
	if ad, ok := findInGroup(addrGrp, avp.AddressData); ok && ad != nil {
		switch s := ad.Data.(type) {
		case datatype.UTF8String:
			v := string(s)
			otherAddr.SmAddressData = &v
		case datatype.OctetString:
			v := string(s)
			otherAddr.SmAddressData = &v
		}
	}

	// Address-Type
	if at, ok := findInGroup(addrGrp, avp.AddressType); ok && at != nil {
		if v, ok := extractNumericAVP(at); ok {
			addrType := smsAddressTypeFromInt(v)
			otherAddr.SmAddressType = &addrType
		}
	}

	return otherAddr
}

// appendSMSStatus populates the SMS delivery status from SMS-Information.
func appendSMSStatus(out *SMSChargingInformation, sms *diam.GroupedAVP) {
	avs, ok := findAvpsInGroup(sms, avp.SMStatus)
	if !ok || len(avs) == 0 {
		return
	}

	if v, ok := extractNumericAVP(&avs[0]); ok {
		status := smsStatusFromInt(v)
		out.SmStatus = &status
	}
}

// appendSMSCAddress populates the SMSC address from SMS-Information.
func appendSMSCAddress(out *SMSChargingInformation, sms *diam.GroupedAVP) {
	if a, ok := findInGroup(sms, avp.SMSCAddress); ok {
		if s, ok := a.Data.(datatype.UTF8String); ok {
			v := string(s)
			out.SmscAddress = &v
		}
	}
}

// extractNumericAVP converts a numeric AVP payload (Integer32/Unsigned32/Enumerated) into a Go int.
func extractNumericAVP(a *diam.AVP) (int, bool) {
	switch s := a.Data.(type) {
	case datatype.Integer32:
		return int(s), true
	case datatype.Unsigned32:
		return int(s), true
	case datatype.Enumerated:
		return int(s), true
	default:
		return 0, false
	}
}

// smsAddressTypeFromInt maps a numeric SM address type into a string representation.
func smsAddressTypeFromInt(val int) string {
	switch val {
	case 0:
		return "E.164"
	case 1:
		return "IPv4"
	case 2:
		return "IPv6"
	case 3:
		return "Email"
	case 4:
		return "SIP_URI"
	case 5:
		return "IMS_PUBLIC_ID"
	default:
		return "E.164"
	}
}

// smsStatusFromInt maps a numeric SM status value into a string representation.
func smsStatusFromInt(val int) string {
	switch val {
	case 0:
		return "DELIVERED"
	case 1:
		return "FORWARDED"
	case 2:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// extractIMSChargingInformation extracts IMS charging information from Service-Information/IMS-Information AVPs.
func extractIMSChargingInformation(m *diam.Message) (*IMSChargingInformation, bool) {
	var gAvp *diam.AVP
	for _, avp_item := range m.AVP {
		if avp_item.Code == avp.ServiceInformation {
			gAvp = avp_item
			break
		}
	}
	if gAvp == nil {
		return nil, false
	}

	g, ok := gAvp.Data.(*diam.GroupedAVP)
	if !ok {
		return nil, false
	}

	ims, ok := findGroup(g, avp.IMSInformation)
	if !ok {
		return nil, false
	}

	chargeInfo := IMSChargingInformation{}

	// Extract calling party addresses
	if a, ok := findInGroup(ims, avp.CalledPartyAddress); ok {
		number := string(a.Data.(datatype.UTF8String))
		chargeInfo.CallingPartyAddresses = []string{number}
	}

	if a, ok := findInGroup(ims, avp.RequestedPartyAddress); ok {
		if number, ok := a.Data.(datatype.UTF8String); ok {
			chargeInfo.RequestedPartyAddress = []string{string(number)}
		}
	}

	if a, ok := findInGroup(ims, avp.RoleOfNode); ok {
		role := int(a.Data.(datatype.Integer32))

		roleOfNode := RoleOfIMSNodeMo
		switch role {
		case 1:
			roleOfNode = RoleOfIMSNodeMt
		case 2:
			roleOfNode = RoleOfIMSNodeMf
		default:
		case 3:
			roleOfNode = RoleOfIMSNodeMo
		}
		chargeInfo.RoleOfNode = &roleOfNode
	}

	return &chargeInfo, true
}

// extractNFIdentification extracts NF identification fields from Origin-Host/Origin-Realm/Visited-PLMN-Id AVPs.
func extractNFIdentification(m *diam.Message) (NFIdentification, bool) {

	nfId := NFIdentification{}

	// Extract NF name
	if nfName, ok := extractAvpString(m, avp.OriginHost); ok {
		nfId.NfName = &nfName
	}

	// Extract NF IPv4 address
	if nfIPv4, ok := extractAvpString(m, avp.OriginRealm); ok {
		nfId.NfIPv4Address = &nfIPv4
	}

	// Extract NF PLMN ID
	if nfPlanId, ok := extractAvpString(m, avp.VisitedPLMNID); ok {
		if len(nfPlanId) > 4 {
			mcc := nfPlanId[0:3]
			mnc := nfPlanId[3:]
			plmnId := PlmnId{
				Mcc: &mcc,
				Mnc: &mnc,
			}
			nfId.NfPLMNID = &plmnId
		}
	}

	smf := NodeFunctionalitySmf
	nfId.NodeFunctionality = &smf
	return nfId, true
}

// extractMultipleUnitUsage extracts per-rating-group usage and requested units from MSCC grouped AVPs.
func extractMultipleUnitUsage(m *diam.Message) ([]MultipleUnitUsage, bool) {
	msccAvps := findMSCCAvps(m)
	if len(msccAvps) == 0 {
		return nil, false
	}

	unitsUsed := make([]MultipleUnitUsage, 0, len(msccAvps))
	for _, cc := range msccAvps {
		g, ok := cc.Data.(*diam.GroupedAVP)
		if !ok || g == nil {
			continue
		}

		usage, ok := buildMultipleUnitUsageFromMSCC(g)
		if !ok {
			continue
		}
		unitsUsed = append(unitsUsed, usage)
	}

	return unitsUsed, true
}

// findMSCCAvps returns all MSCC (Multiple-Services-Credit-Control) AVPs from a Diameter message.
func findMSCCAvps(m *diam.Message) []*diam.AVP {
	if m == nil {
		return nil
	}
	var mscc []*diam.AVP
	for _, a := range m.AVP {
		if a != nil && a.Code == avp.MultipleServicesCreditControl {
			mscc = append(mscc, a)
		}
	}
	return mscc
}

// buildMultipleUnitUsageFromMSCC builds a MultipleUnitUsage instance from a single MSCC grouped AVP.
func buildMultipleUnitUsageFromMSCC(g *diam.GroupedAVP) (MultipleUnitUsage, bool) {
	usage := MultipleUnitUsage{}

	rg := int64(parseRatingGroup(g))
	usage.RatingGroup = &rg

	if container, ok := extractUsedUnitContainer(g); ok {
		usage.UsedUnitContainer = []UsedUnitContainer{container}
	} else {
		return usage, false
	}

	if requestedUnit, ok := extractRequestUnits(g); ok {
		usage.RequestedUnit = requestedUnit
	}

	return usage, true
}

// parseRatingGroup extracts the Rating-Group value from an MSCC grouped AVP.
func parseRatingGroup(g *diam.GroupedAVP) int {
	if g == nil {
		return 0
	}
	if rgAvp, ok := findInGroup(g, avp.RatingGroup); ok && rgAvp != nil {
		if v, ok := extractNumericAVP(rgAvp); ok {
			return v
		}
	}
	return 0
}

// extractRequestUnits extracts RequestedUnit values from a Requested-Service-Unit grouped AVP.
func extractRequestUnits(g *diam.GroupedAVP) (*RequestedUnit, bool) {
	if a, ok := findInGroup(g, avp.RequestedServiceUnit); ok {
		reqGrp, _ := a.Data.(*diam.GroupedAVP)

		requestedUnits := RequestedUnit{}
		if u, ok := extractUnits(reqGrp, avp.CCTime); ok {
			t := int(u)
			requestedUnits.Time = &t
		}
		if u, ok := extractUnits(reqGrp, avp.CCTotalOctets); ok {
			requestedUnits.TotalVolume = &u
		}
		if u, ok := extractUnits(reqGrp, avp.CCInputOctets); ok {
			requestedUnits.UplinkVolume = &u
		}
		if u, ok := extractUnits(reqGrp, avp.CCOutputOctets); ok {
			requestedUnits.DownlinkVolume = &u
		}
		if u, ok := extractUnits(reqGrp, avp.CCServiceSpecificUnits); ok {
			requestedUnits.ServiceSpecificUnits = &u
		}

		return &requestedUnits, true
	}

	return nil, true
}

// extractUsedUnitContainer extracts UsedUnitContainer values from a Used-Service-Unit grouped AVP.
func extractUsedUnitContainer(g *diam.GroupedAVP) (UsedUnitContainer, bool) {

	container := UsedUnitContainer{}
	success := false

	if a, ok := findInGroup(g, avp.UsedServiceUnit); ok {
		usedGrp, _ := a.Data.(*diam.GroupedAVP)

		if u, ok := extractUnits(usedGrp, avp.CCTime); ok {
			t := int(u)
			container.Time = &t
			success = true
		}
		if u, ok := extractUnits(usedGrp, avp.CCTotalOctets); ok {
			container.TotalVolume = &u
			success = true
		}
		if u, ok := extractUnits(usedGrp, avp.CCInputOctets); ok {
			container.UplinkVolume = &u
			success = true
		}
		if u, ok := extractUnits(usedGrp, avp.CCOutputOctets); ok {
			container.DownlinkVolume = &u
			success = true
		}
		if u, ok := extractUnits(usedGrp, avp.CCServiceSpecificUnits); ok {
			container.ServiceSpecificUnits = &u
			success = true
		}

		//Always true for DCCaNchf
		reclaimable := true
		container.Reclaimable = &reclaimable
	}

	return container, success
}

// extractUnits reads a numeric unit AVP from a grouped container and returns it as an int64.
func extractUnits(g *diam.GroupedAVP, code uint32) (int64, bool) {

	if t, ok := findInGroup(g, code); ok {
		switch u := t.Data.(type) {
		case datatype.Integer32:
			usage := int(u)
			return int64(usage), true
		case datatype.Unsigned32:
			usage := int(u)
			return int64(usage), true
		case datatype.Integer64:
			usage := int64(u)
			return int64(usage), true
		case datatype.Unsigned64:
			usage := int64(u)
			return int64(usage), true
		}
	}

	return 0, false
}

// findGroup returns the first grouped AVP with the given code from within a grouped AVP.
func findGroup(g *diam.GroupedAVP, code uint32) (*diam.GroupedAVP, bool) {
	for _, a := range g.AVP {
		if a != nil && a.Code == code && a.Data.Type() == diam.GroupedAVPType {
			return a.Data.(*diam.GroupedAVP), true
		}
	}
	return nil, false
}

// findInGroup returns the first AVP with the given code from within a grouped AVP.
func findInGroup(g *diam.GroupedAVP, code uint32) (*diam.AVP, bool) {
	for _, a := range g.AVP {
		if a != nil && a.Code == code {
			return a, true
		}
	}
	return nil, false
}

// findAvpsInGroup returns all AVPs with the given code from within a grouped AVP.
func findAvpsInGroup(g *diam.GroupedAVP, code uint32) ([]diam.AVP, bool) {
	avps := []diam.AVP{}
	for _, a := range g.AVP {
		if a != nil && a.Code == code {
			avps = append(avps, *a)
		}
	}
	return avps, len(avps) != 0
}

// extractAvpString extracts a UTF8String or OctetString AVP value from a Diameter message.
func extractAvpString(m *diam.Message, code uint32) (string, bool) {
	var a *diam.AVP
	for _, avp_item := range m.AVP {
		if avp_item.Code == code {
			a = avp_item
			break
		}
	}
	if a == nil {
		return "", false
	}

	switch s := a.Data.(type) {
	case datatype.UTF8String:
		return string(s), true
	case datatype.OctetString:
		return string(s), true
	}

	return "", false
}

// extractAvpInteger extracts an Integer32/Unsigned32/Enumerated AVP value from a Diameter message.
func extractAvpInteger(m *diam.Message, code uint32) (int, bool) {
	var a *diam.AVP
	for _, avp_item := range m.AVP {
		if avp_item.Code == code {
			a = avp_item
			break
		}
	}
	if a == nil {
		return 0, false
	}

	ok := false
	result := 0
	switch s := a.Data.(type) {
	case datatype.Integer32:
		result = int(s)
		ok = true
	case datatype.Unsigned32:
		result = int(s)
		ok = true
	case datatype.Enumerated:
		result = int(s)
		ok = true
	}

	return result, ok
}

// extractTimestamp extracts an Event-Timestamp AVP and converts it to LocalDateTime.
func extractTimestamp(m *diam.Message, code uint32) (LocalDateTime, bool) {
	var a *diam.AVP
	for _, avp_item := range m.AVP {
		if avp_item.Code == code {
			a = avp_item
			break
		}
	}
	if a == nil {
		return LocalDateTime{}, false
	}

	if t, ok := a.Data.(datatype.Time); ok {
		return LocalDateTime(t), true
	}

	return LocalDateTime{}, false
}

// extractSubscriberIdentifier extracts the MSISDN (END_USER_E164) from Subscription-Id grouped AVPs.
func extractSubscriberIdentifier(m *diam.Message) *string {
	// 443 Subscription-Id (Grouped, can appear multiple times)
	var subs []*diam.AVP
	for _, avp_item := range m.AVP {
		if avp_item.Code == avp.SubscriptionID {
			subs = append(subs, avp_item)
		}
	}
	if len(subs) == 0 {
		return nil
	}

	for _, sub := range subs {
		g, ok := sub.Data.(*diam.GroupedAVP)
		if !ok || g == nil {
			continue
		}

		// 450 Subscription-Id-Type
		tAvp, ok := findInGroup(g, avp.SubscriptionIDType)
		if !ok {
			continue
		}
		t, ok := tAvp.Data.(datatype.Enumerated)
		if !ok {
			continue
		}
		if uint32(t) != 0 { // END_USER_E164
			continue
		}

		// 444 Subscription-Id-Data
		dAvp, ok := findInGroup(g, avp.SubscriptionIDData)
		if !ok {
			continue
		}

		if s, ok := dAvp.Data.(datatype.UTF8String); ok {
			val := string(s)
			return &val
		}
	}

	return nil
}

package nchf

import (
	"testing"
	"time"

	"github.com/fiorix/go-diameter/v4/diam"
	"github.com/fiorix/go-diameter/v4/diam/avp"
	"github.com/fiorix/go-diameter/v4/diam/datatype"
	"github.com/fiorix/go-diameter/v4/diam/dict"
)

func TestExtractAvpString(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)
	m.AddAVP(diam.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String("session-123")))

	val, ok := extractAvpString(m, avp.SessionID)
	if !ok {
		t.Fatal("expected ok")
	}
	if val != "session-123" {
		t.Errorf("expected session-123, got %s", val)
	}

	_, ok = extractAvpString(m, avp.OriginHost)
	if ok {
		t.Error("expected !ok for missing AVP")
	}
}

func TestExtractAvpInteger(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)
	m.AddAVP(diam.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(5)))

	val, ok := extractAvpInteger(m, avp.CCRequestNumber)
	if !ok {
		t.Fatal("expected ok")
	}
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}

	_, ok = extractAvpInteger(m, avp.CCRequestType)
	if ok {
		t.Error("expected !ok for missing AVP")
	}
}

func TestExtractTimestamp(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)
	now := time.Now().Truncate(time.Second)
	m.AddAVP(diam.NewAVP(avp.EventTimestamp, avp.Mbit, 0, datatype.Time(now)))

	ts, ok := extractTimestamp(m, avp.EventTimestamp)
	if !ok {
		t.Fatal("expected ok")
	}
	if !ts.Time().Equal(now) {
		t.Errorf("expected %v, got %v", now, ts.Time())
	}
}

func TestFindInGroup(t *testing.T) {
	g := &diam.GroupedAVP{
		AVP: []*diam.AVP{
			{
				Code: avp.SubscriptionIDType,
				Data: datatype.Enumerated(0),
			},
		},
	}

	a, ok := findInGroup(g, avp.SubscriptionIDType)
	if !ok {
		t.Fatal("expected ok")
	}
	if a.Code != avp.SubscriptionIDType {
		t.Errorf("expected code %d, got %d", avp.SubscriptionIDType, a.Code)
	}

	_, ok = findInGroup(g, avp.SubscriptionIDData)
	if ok {
		t.Error("expected !ok for missing AVP")
	}
}

func TestExtractSubscriberIdentifier(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)

	// Subscription-Id 443
	//   Subscription-Id-Type 450 = 0 (END_USER_E164)
	//   Subscription-Id-Data 444 = "123456789"
	subId := diam.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.SubscriptionIDType, avp.Mbit, 0, datatype.Enumerated(0)),
			diam.NewAVP(avp.SubscriptionIDData, avp.Mbit, 0, datatype.UTF8String("123456789")),
		},
	})
	m.AddAVP(subId)

	res := extractSubscriberIdentifier(m)
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if *res != "123456789" {
		t.Errorf("expected 123456789, got %s", *res)
	}

	// Test with wrong type
	m2 := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)
	subId2 := diam.NewAVP(avp.SubscriptionID, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.SubscriptionIDType, avp.Mbit, 0, datatype.Enumerated(1)), // IMSI
			diam.NewAVP(avp.SubscriptionIDData, avp.Mbit, 0, datatype.UTF8String("123456789")),
		},
	})
	m2.AddAVP(subId2)
	res2 := extractSubscriberIdentifier(m2)
	if res2 != nil {
		t.Errorf("expected nil result for IMSI, got %s", *res2)
	}
}

func TestExtractNFIdentification(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)
	m.AddAVP(diam.NewAVP(avp.OriginHost, avp.Mbit, 0, datatype.UTF8String("smf-1")))
	m.AddAVP(diam.NewAVP(avp.OriginRealm, avp.Mbit, 0, datatype.UTF8String("10.0.0.1")))
	m.AddAVP(diam.NewAVP(avp.VisitedPLMNID, avp.Mbit, 0, datatype.OctetString("20801")))

	nfId, ok := extractNFIdentification(m)
	if !ok {
		t.Fatal("expected ok")
	}
	if *nfId.NfName != "smf-1" {
		t.Errorf("expected smf-1, got %s", *nfId.NfName)
	}
	if *nfId.NfIPv4Address != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", *nfId.NfIPv4Address)
	}
	if nfId.NfPLMNID == nil || *nfId.NfPLMNID.Mcc != "208" || *nfId.NfPLMNID.Mnc != "01" {
		t.Errorf("expected PLMN 208/01, got %+v", nfId.NfPLMNID)
	}
}

func TestExtractMultipleUnitUsage(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)

	// Multiple-Services-Credit-Control 456
	//   Rating-Group 432 = 10
	//   Used-Service-Unit 446
	//     CC-Time 420 = 60
	mscc := diam.NewAVP(avp.MultipleServicesCreditControl, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.RatingGroup, avp.Mbit, 0, datatype.Unsigned32(10)),
			diam.NewAVP(avp.UsedServiceUnit, avp.Mbit, 0, &diam.GroupedAVP{
				AVP: []*diam.AVP{
					diam.NewAVP(avp.CCTime, avp.Mbit, 0, datatype.Unsigned32(60)),
				},
			}),
		},
	})
	m.AddAVP(mscc)

	units, ok := extractMultipleUnitUsage(m)
	if !ok {
		t.Fatal("expected ok")
	}
	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}
	if *units[0].RatingGroup != 10 {
		t.Errorf("expected RG 10, got %d", *units[0].RatingGroup)
	}
	if len(units[0].UsedUnitContainer) != 1 {
		t.Fatalf("expected 1 container, got %d", len(units[0].UsedUnitContainer))
	}
	if *units[0].UsedUnitContainer[0].Time != 60 {
		t.Errorf("expected time 60, got %d", *units[0].UsedUnitContainer[0].Time)
	}
}

func TestExtractSMSChargingInformation(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)

	// Service-Information 455
	//   SMS-Information 2000
	//     Originator-Interface 2001 = "orig-supi"
	//     SMS-Message-Type 2002 = "SUBMIT"
	smsInfoAVP := diam.NewAVP(avp.SMSInformation, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.OriginatorInterface, avp.Mbit, 0, datatype.UTF8String("orig-supi")),
			diam.NewAVP(avp.MessageType, avp.Mbit, 0, datatype.UTF8String("SUBMIT")),
		},
	})
	m.AddAVP(diam.NewAVP(avp.ServiceInformation, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{smsInfoAVP},
	}))

	sms, ok := extractSMSChargingInformation(m)
	if !ok {
		t.Fatal("expected ok")
	}
	if *sms.OriginatorInfo.OriginatorSUPI != "orig-supi" {
		t.Errorf("expected orig-supi, got %s", *sms.OriginatorInfo.OriginatorSUPI)
	}
	if *sms.SmMessageType != "SUBMIT" {
		t.Errorf("expected SUBMIT, got %s", *sms.SmMessageType)
	}
}

func TestExtractIMSChargingInformation(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)

	// Service-Information 455
	//   IMS-Information 876
	//     Role-Of-Node 829 = 0 (MO)
	//     Called-Party-Address 832 = "tel:+123456"
	imsInfoAVP := diam.NewAVP(avp.IMSInformation, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.RoleOfNode, avp.Mbit, 0, datatype.Integer32(0)),
			diam.NewAVP(avp.CalledPartyAddress, avp.Mbit, 0, datatype.UTF8String("tel:+123456")),
		},
	})
	m.AddAVP(diam.NewAVP(avp.ServiceInformation, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{imsInfoAVP},
	}))

	ims, ok := extractIMSChargingInformation(m)
	if !ok {
		t.Fatal("expected ok")
	}
	if *ims.RoleOfNode != RoleOfIMSNodeMo {
		t.Errorf("expected MO, got %v", *ims.RoleOfNode)
	}
	if len(ims.CallingPartyAddresses) == 0 || ims.CallingPartyAddresses[0] != "tel:+123456" {
		t.Errorf("expected tel:+123456, got %v", ims.CallingPartyAddresses)
	}
}

func TestAvpToNchfRequest(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)
	m.AddAVP(diam.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String("sess-1")))
	m.AddAVP(diam.NewAVP(avp.CCRequestNumber, avp.Mbit, 0, datatype.Unsigned32(1)))
	m.AddAVP(diam.NewAVP(avp.CCRequestType, avp.Mbit, 0, datatype.Unsigned32(4))) // EVENT_REQUEST

	req, err := AvpToNchfRequest(m)
	if err != nil {
		t.Fatalf("AvpToNchfRequest failed: %v", err)
	}

	if req.ChargingId == nil || *req.ChargingId != "sess-1" {
		t.Errorf("expected sess-1, got %v", req.ChargingId)
	}
	if req.InvocationSequenceNumber == nil || *req.InvocationSequenceNumber != 1 {
		t.Errorf("expected 1, got %v", req.InvocationSequenceNumber)
	}
	if req.OneTimeEvent == nil || !*req.OneTimeEvent {
		t.Errorf("expected OneTimeEvent to be true, got %v", req.OneTimeEvent)
	}
}

func TestExtractPDUSessionChargingInformation(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)
	m.AddAVP(diam.NewAVP(avp.SessionID, avp.Mbit, 0, datatype.UTF8String("pdu-sess-1")))

	psInfo := diam.NewAVP(avp.PSInformation, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(avp.TGPPChargingID, avp.Mbit, 0, datatype.OctetString("charging-id")),
		},
	})
	m.AddAVP(diam.NewAVP(avp.ServiceInformation, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{psInfo},
	}))

	pdu, _ := extractPDUSessionChargingInformation(m)
	if pdu == nil {
		t.Fatal("expected non-nil pdu")
	}
	if *pdu.ChargingId != "pdu-sess-1" {
		t.Errorf("expected pdu-sess-1, got %s", *pdu.ChargingId)
	}
}

func TestExtractNEFChargingInformation(t *testing.T) {
	m := diam.NewMessage(diam.CreditControl, diam.RequestFlag, 0, 0, 0, dict.Default)

	ussdInfo := diam.NewAVP(AVP_USSD_INFORMATION, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{
			diam.NewAVP(AVP_USSD_STRING, avp.Mbit, 0, datatype.UTF8String("ussd-code")),
		},
	})
	m.AddAVP(diam.NewAVP(avp.ServiceInformation, avp.Mbit, 0, &diam.GroupedAVP{
		AVP: []*diam.AVP{ussdInfo},
	}))

	nef, _ := extractNEFChargingInformation(m)
	if nef == nil {
		t.Fatal("expected non-nil nef")
	}
	if *nef.ExternalGroupIdentifier != "ussd-code" {
		t.Errorf("expected ussd-code, got %s", *nef.ExternalGroupIdentifier)
	}
}

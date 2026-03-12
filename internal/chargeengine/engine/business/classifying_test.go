package business

import (
	"go-ocs/internal/chargeengine/engine"
	"go-ocs/internal/chargeengine/engine/providers/carriers"
	"go-ocs/internal/chargeengine/model"
	"go-ocs/internal/charging"
	"go-ocs/internal/common"
	"go-ocs/internal/logging"
	"go-ocs/internal/nchf"
	"go-ocs/internal/store/sqlc"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"gopkg.in/yaml.v3"
)

type mockInfra struct {
	plan *model.Plan
}

func (m *mockInfra) FindSubscriber(msisdn string) (*model.Subscriber, error) {
	return nil, nil
}

func (m *mockInfra) FetchClassificationPlan() (*model.Plan, error) {
	return m.plan, nil
}

func (m *mockInfra) FetchCarrierContainer() (*carriers.CarrierContainer, error) {
	return nil, nil
}

func (m *mockInfra) FindCarrierByMccMnc(mcc string, mnc string) (*sqlc.Carrier, error) {
	return nil, nil
}

func (m *mockInfra) FindCarrierBySource(mcc string, mnc string) string {
	return "TESTSOURCE"
}

func (m *mockInfra) FindNumberPlan(number string) (*sqlc.AllNumbersRow, error) {
	return nil, nil
}

func (m *mockInfra) FindCarrierByDestination(number string) string {
	return "LOCAL"
}

func (m *mockInfra) FindRatingPlan(uuid uuid.UUID) (*model.RatePlan, error) {
	return nil, nil
}

func loadPlan(t *testing.T) *model.Plan {
	// Adjust path to be relative to project root
	data, err := os.ReadFile("../../../../internal/chargeengine/engine/providers/classificationplan/classificationPlan.yaml")
	if err != nil {
		t.Fatalf("Failed to read plan file: %v", err)
	}

	content := string(data)
	// Replace complex expressions with simple ones that mockInfra can handle
	// Use more specific replacements to avoid breaking YAML
	content = strings.ReplaceAll(content, "sourceByMccMnc(Req.NfConsumerIdentification.NfPLMNID.Mcc, Req.NfConsumerIdentification.NfPLMNID.Mnc)", "'TESTSOURCE'")
	content = strings.ReplaceAll(content, "sourceByMccMnc(Req.NfConsumerIdentification.NfPLMNID.Mcc,Req.NfConsumerIdentification.NfPLMNID.Mnc)", "'TESTSOURCE'")
	content = strings.ReplaceAll(content, "sourceByMccMnc(NfConsumerIdentification.NfPLMNID.Mcc, NfConsumerIdentification.NfPLMNID.Mnc)", "'TESTSOURCE'")
	content = strings.ReplaceAll(content, "Info.RecipientInfo[0].RecipientOtherAddress.SmAddressData)", "'LOCAL'")

	// Additional replacements for common CEL expressions in classificationPlan.yaml
	content = strings.ReplaceAll(content, "serviceDirection()", "'MO'")
	content = strings.ReplaceAll(content, "serviceCategory()", "'NORMAL'")
	content = strings.ReplaceAll(content, "serviceTypeRule: \"OneTimeEvent\"", "serviceTypeRule: \"Req.OneTimeEvent == true\"")
	content = strings.ReplaceAll(content, "serviceTypeRule: \"not OneTimeEvent\"", "serviceTypeRule: \"Req.OneTimeEvent == false\"")
	content = strings.ReplaceAll(content, "serviceTypeRule: OneTimeEvent", "serviceTypeRule: \"Req.OneTimeEvent == true\"")
	content = strings.ReplaceAll(content, "serviceTypeRule: not OneTimeEvent", "serviceTypeRule: \"Req.OneTimeEvent == false\"")

	plan := &model.Plan{}
	err = yaml.Unmarshal([]byte(content), plan)
	if err != nil {
		t.Fatalf("Failed to unmarshal plan: %v", err)
	}

	// Post-processing similar to classificationprovider.go
	for i := range plan.ServiceTypes {
		st := &plan.ServiceTypes[i]
		st.ServiceWindowMap = make(map[string]struct{})
		if st.ServiceWindows != nil {
			for _, n := range st.ServiceWindows {
				st.ServiceWindowMap[n] = struct{}{}
			}
		}
	}

	return plan
}

func parseTime(s string) common.LocalTime {
	t, _ := time.Parse("15:04", s)
	return common.LocalTime{Time: t}
}

func ptrStr(s string) *string { return &s }
func ptrInt(i int) *int64     { j := int64(i); return &j }

func TestClassifyService_Voice(t *testing.T) {
	logging.Bootstrap()
	plan := loadPlan(t)
	infra := &mockInfra{plan: plan}

	rg := 1
	req := nchf.NewChargingDataRequest()
	moRole := nchf.RoleOfIMSNodeMo
	req.ImsChargingInformation = &nchf.IMSChargingInformation{
		RoleOfNode:         &moRole,
		CalledPartyAddress: ptrStr("tel:+123456789"),
	}
	req.NfConsumerIdentification = &nchf.NFIdentification{
		NfPLMNID: &nchf.PlmnId{
			Mcc: ptrStr("234"),
			Mnc: ptrStr("15"),
		},
	}
	req.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{
			RatingGroup: ptrInt(rg),
		},
	}

	dc := &engine.ChargingContext{
		Infra:   infra,
		Request: req,
	}

	classifications, err := ClassifyService(dc)
	if err != nil {
		t.Fatalf("ClassifyService failed: %v", err)
	}

	for _, c := range classifications {
		logging.Debug("Classification", "rateKey", c.Ratekey.String())
	}

	if len(classifications) != 1 {
		t.Errorf("Expected 1 classification, got %d", len(classifications))
	}

	c, ok := classifications[int64(rg)]
	if !ok {
		t.Fatalf("Classification for rating group 1 not found")
	}

	expectedServiceType := "VOICE"
	if c.Ratekey.ServiceType != expectedServiceType {
		t.Errorf("Expected ServiceType %s, got %s", expectedServiceType, c.Ratekey.ServiceType)
	}

	// findCarrierBySource returns "TESTSOURCE"
	if c.Ratekey.SourceType != "TESTSOURCE" {
		t.Errorf("Expected SourceType TESTSOURCE, got %s", c.Ratekey.SourceType)
	}

	// serviceDirection() returns MO by default in current mock
	if c.Ratekey.ServiceDirection != charging.MO {
		t.Errorf("Expected ServiceDirection MO, got %s", c.Ratekey.ServiceDirection)
	}

	if c.UnitType != charging.SECONDS {
		t.Errorf("Expected UnitType SECONDS, got %s", c.UnitType)
	}
}

func TestClassifyService_Data(t *testing.T) {
	logging.Bootstrap()
	plan := loadPlan(t)
	infra := &mockInfra{plan: plan}

	rg1 := 123
	rg2 := 10010010
	req := nchf.NewChargingDataRequest()
	req.NfConsumerIdentification = &nchf.NFIdentification{
		NfPLMNID: &nchf.PlmnId{
			Mcc: ptrStr("234"),
			Mnc: ptrStr("15"),
		},
	}
	req.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{RatingGroup: ptrInt(rg1)},
		{RatingGroup: ptrInt(rg2)},
	}

	// PDU type request
	req.PduSessionChargingInformation = &nchf.PDUSessionChargingInformation{}

	dc := &engine.ChargingContext{
		Infra:   infra,
		Request: req,
	}

	classifications, err := ClassifyService(dc)
	if err != nil {
		t.Fatalf("ClassifyService failed: %v", err)
	}

	for _, c := range classifications {
		logging.Debug("Classification", "rateKey", c.Ratekey.String())
	}

	if len(classifications) != 2 {
		t.Errorf("Expected 2 classifications, got %d", len(classifications))
	}

	c123, ok1 := classifications[int64(rg1)]
	if !ok1 || c123.Ratekey.ServiceType != "DATA" {
		t.Errorf("Expected ServiceType DATA for RG 123, got %s", c123.Ratekey.ServiceType)
	}

	cFree, ok2 := classifications[int64(rg2)]
	if !ok2 || cFree.Ratekey.ServiceType != "DATA" {
		t.Errorf("Expected ServiceType DATA for RG 10010010, got %s", cFree.Ratekey.ServiceType)
	}
}

func TestGetServiceWindow(t *testing.T) {
	sws := map[string]model.ServiceWindow{
		"PEAK": {
			StartTime: parseTime("08:00"),
			EndTime:   parseTime("20:00"),
		},
		"OFFPEAK": {
			StartTime: parseTime("20:01"),
			EndTime:   parseTime("07:59"),
		},
	}
	valid := map[string]struct{}{
		"PEAK":    {},
		"OFFPEAK": {},
	}

	// 10:00 is PEAK
	now := time.Date(0, 1, 1, 10, 0, 0, 0, time.UTC)
	win := GetServiceWindow(sws, valid, "STANDARD", now)
	if win != "PEAK" {
		t.Errorf("Expected PEAK, got %s", win)
	}
}

func TestClassifyService_SMS(t *testing.T) {
	logging.Bootstrap()
	plan := loadPlan(t)
	infra := &mockInfra{plan: plan}

	rg := 10
	req := nchf.NewChargingDataRequest()
	req.NfConsumerIdentification = &nchf.NFIdentification{
		NfPLMNID: &nchf.PlmnId{
			Mcc: ptrStr("234"),
			Mnc: ptrStr("15"),
		},
	}
	req.SmsChargingInformation = &nchf.SMSChargingInformation{
		RecipientInfo: []nchf.RecipientInfo{
			{
				RecipientOtherAddress: &nchf.SMAddressInfo{
					SmAddressData: ptrStr("555555"),
				},
			},
		},
	}
	req.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{RatingGroup: ptrInt(rg)},
	}

	dc := &engine.ChargingContext{
		Infra:   infra,
		Request: req,
	}

	classifications, err := ClassifyService(dc)
	if err != nil {
		t.Fatalf("ClassifyService failed: %v", err)
	}

	for _, c := range classifications {
		logging.Debug("Classification", "rateKey", c.Ratekey.String())
	}

	c, ok := classifications[int64(rg)]
	if !ok {
		t.Fatalf("Classification for rating group %d not found", rg)
	}

	if c.Ratekey.ServiceType != "SMS" {
		t.Errorf("Expected ServiceType SMS, got %s", c.Ratekey.ServiceType)
	}

	if c.UnitType != charging.UNITS {
		t.Errorf("Expected UnitType UNITS, got %s", c.UnitType)
	}
}

func TestClassifyService_USSD(t *testing.T) {
	logging.Bootstrap()

	plan := loadPlan(t)
	infra := &mockInfra{plan: plan}

	rg := 20
	req := nchf.NewChargingDataRequest()
	req.NfConsumerIdentification = &nchf.NFIdentification{
		NfPLMNID: &nchf.PlmnId{
			Mcc: ptrStr("234"),
			Mnc: ptrStr("15"),
		},
	}
	oneTime := true
	req.OneTimeEvent = &oneTime
	req.NefChargingInformation = &nchf.NEFChargingInformation{
		ExternalGroupIdentifier: ptrStr("*100#"),
	}
	req.MultipleUnitUsage = []nchf.MultipleUnitUsage{
		{RatingGroup: ptrInt(rg)},
	}

	dc := &engine.ChargingContext{
		Infra:   infra,
		Request: req,
	}

	classifications, err := ClassifyService(dc)
	if err != nil {
		t.Fatalf("ClassifyService failed (OneTimeEvent=true): %v", err)
	}

	for _, c := range classifications {
		logging.Debug("Classification", "rateKey", c.Ratekey.String())
	}

	c, ok := classifications[int64(rg)]
	if !ok {
		// Debug the plan
		for _, st := range plan.ServiceTypes {
			t.Logf("Plan ServiceType: %s, ChargingInformation: %s, Rule: %s", st.ServiceType, st.ChargingInformation, st.ServiceTypeRule)
		}
		t.Fatalf("Classification for rating group %d not found", rg)
	}

	if c.Ratekey.ServiceType != "USSD1" {
		t.Errorf("Expected ServiceType USSD1, got %s", c.Ratekey.ServiceType)
	}
	if c.UnitType != charging.UNITS {
		t.Errorf("Expected UnitType UNITS, got %s", c.UnitType)
	}

	// For USSD2 (not OneTimeEvent)
	oneTime = false
	req.OneTimeEvent = &oneTime
	classifications, err = ClassifyService(dc)

	for _, c := range classifications {
		logging.Debug("Classification", "rateKey", c.Ratekey.String())
	}
	if err != nil {
		t.Fatalf("ClassifyService failed (OneTimeEvent=false): %v", err)
	}
	c, ok = classifications[int64(rg)]
	if !ok {
		t.Fatalf("Classification for rating group %d not found", rg)
	}
	if c.Ratekey.ServiceType != "USSD2" {
		t.Errorf("Expected ServiceType USSD2, got %s", c.Ratekey.ServiceType)
	}
	if c.UnitType != charging.SECONDS {
		t.Errorf("Expected UnitType SECONDS, got %s", c.UnitType)
	}
}

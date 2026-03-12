package numberplan

import (
	"context"
	"errors"
	"go-ocs/internal/store/sqlc"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNumberPlanContainer_reloadNumbers(t *testing.T) {
	mockData := []sqlc.AllNumbersRow{
		{Plmn: "65501", NumberRange: "2783", DestinationGroup: "ZA_MTN"},
		{Plmn: "65502", NumberRange: "2784", DestinationGroup: "ZA_VODACOM"},
	}

	loader := func(ctx context.Context) ([]sqlc.AllNumbersRow, error) {
		return mockData, nil
	}

	container := &NumberPlanContainer{
		loader: loader,
	}

	container.reloadNumbers(context.Background(), "27")

	assert.Equal(t, 2, len(container.numbers))
	assert.Equal(t, "ZA_MTN", container.numbers["65501"].DestinationGroup)
	assert.Equal(t, "ZA_VODACOM", container.numbers["65502"].DestinationGroup)
	assert.Equal(t, "27", container.nationalDialCode)
}

func TestNumberPlanContainer_reloadNumbers_Error(t *testing.T) {
	loader := func(ctx context.Context) ([]sqlc.AllNumbersRow, error) {
		return nil, errors.New("load error")
	}

	container := &NumberPlanContainer{
		loader:  loader,
		numbers: make(map[string]sqlc.AllNumbersRow),
	}
	container.numbers["old"] = sqlc.AllNumbersRow{Plmn: "old"}

	container.reloadNumbers(context.Background(), "27")

	// Should not overwrite on error
	assert.Equal(t, 1, len(container.numbers))
	assert.Equal(t, "old", container.numbers["old"].Plmn)
}

func TestNumberPlanContainer_FindNumberPlan(t *testing.T) {
	mockData := map[string]sqlc.AllNumbersRow{
		"65501": {Plmn: "65501", NumberRange: "2783", DestinationGroup: "ZA_MTN"},
		"65502": {Plmn: "65502", NumberRange: "2784", DestinationGroup: "ZA_VODACOM"},
		"65510": {Plmn: "65510", NumberRange: "27831", DestinationGroup: "ZA_MTN_SPECIFIC"},
	}

	container := &NumberPlanContainer{
		numbers:          mockData,
		nationalDialCode: "27",
	}

	tests := []struct {
		name          string
		number        string
		expectedGroup string
		expectError   bool
	}{
		{
			name:          "Exact prefix match",
			number:        "27841234567",
			expectedGroup: "ZA_VODACOM",
			expectError:   false,
		},
		{
			name:          "Longest prefix match",
			number:        "27831234567",
			expectedGroup: "ZA_MTN_SPECIFIC", // matches 27831 instead of 2783
			expectError:   false,
		},
		{
			name:          "National dial code conversion",
			number:        "0841234567",
			expectedGroup: "ZA_VODACOM", // 084 -> 2784
			expectError:   false,
		},
		{
			name:          "No match",
			number:        "27851234567",
			expectedGroup: "",
			expectError:   true,
		},
		{
			name:          "Empty number",
			number:        "",
			expectedGroup: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := container.FindNumberPlan(tt.number)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, match)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, match)
				assert.Equal(t, tt.expectedGroup, match.DestinationGroup)
			}
		})
	}
}

func TestNumberPlanContainer_FindCarrierByDestination(t *testing.T) {
	mockData := map[string]sqlc.AllNumbersRow{
		"65501": {Plmn: "65501", NumberRange: "2783", DestinationGroup: "ZA_MTN"},
	}

	container := &NumberPlanContainer{
		numbers:          mockData,
		nationalDialCode: "27",
	}

	assert.Equal(t, "ZA_MTN", container.FindCarrierByDestination("2783123"))
	assert.Equal(t, "?", container.FindCarrierByDestination("2785123"))
}

func TestNumberPlanContainer_Shutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func() {
		shutdownCalled = true
	}

	container := &NumberPlanContainer{
		shutdown: shutdown,
	}

	container.Shutdown()
	assert.True(t, shutdownCalled)
}

func TestNumberPlanContainer_NotLoaded(t *testing.T) {
	container := &NumberPlanContainer{}
	match, err := container.FindNumberPlan("2783")
	assert.Error(t, err)
	assert.Nil(t, match)
	assert.Equal(t, "number plan list not loaded", err.Error())
}

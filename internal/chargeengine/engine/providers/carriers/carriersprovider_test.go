package carriers

import (
	"context"
	"go-ocs/internal/store/sqlc"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestCarrierContainer_reloadCarriers(t *testing.T) {
	mockCarriers := []sqlc.Carrier{
		{
			Plmn:        "23401",
			Mcc:         "234",
			Mnc:         pgtype.Text{String: "01", Valid: true},
			SourceGroup: "UK_VODA",
		},
		{
			Plmn:        "23402",
			Mcc:         "234",
			Mnc:         pgtype.Text{String: "02", Valid: true},
			SourceGroup: "UK_O2",
		},
	}

	loader := func(ctx context.Context) ([]sqlc.Carrier, error) {
		return mockCarriers, nil
	}

	container := &CarrierContainer{
		loader: loader,
	}

	container.reloadCarriers(context.Background())

	assert.Len(t, container.carriers, 2)
	assert.Equal(t, "UK_VODA", container.carriers["23401"].SourceGroup)
	assert.Equal(t, "UK_O2", container.carriers["23402"].SourceGroup)
}

func TestCarrierContainer_FindCarrierByMccMnc(t *testing.T) {
	mockCarriers := map[string]sqlc.Carrier{
		"23401": {
			Plmn:        "23401",
			Mcc:         "234",
			Mnc:         pgtype.Text{String: "01", Valid: true},
			SourceGroup: "UK_VODA",
		},
		"234": {
			Plmn:        "234",
			Mcc:         "234",
			SourceGroup: "UK_GENERIC",
		},
	}

	container := &CarrierContainer{
		carriers: mockCarriers,
	}

	// Test Exact Match
	carrier, err := container.FindCarrierByMccMnc("234", "01")
	assert.NoError(t, err)
	assert.NotNil(t, carrier)
	assert.Equal(t, "UK_VODA", carrier.SourceGroup)

	// Test Fallback Match (MCC only)
	carrier, err = container.FindCarrierByMccMnc("234", "99")
	assert.NoError(t, err)
	assert.NotNil(t, carrier)
	assert.Equal(t, "UK_GENERIC", carrier.SourceGroup)

	// Test Not Found
	carrier, err = container.FindCarrierByMccMnc("505", "01")
	assert.Error(t, err)
	assert.Nil(t, carrier)

	// Test Carriers not loaded
	emptyContainer := &CarrierContainer{}
	carrier, err = emptyContainer.FindCarrierByMccMnc("234", "01")
	assert.Error(t, err)
	assert.Equal(t, "carrier list not loaded", err.Error())
}

func TestCarrierContainer_FindCarrierBySource(t *testing.T) {
	mockCarriers := map[string]sqlc.Carrier{
		"23401": {
			Plmn:        "23401",
			Mcc:         "234",
			Mnc:         pgtype.Text{String: "01", Valid: true},
			SourceGroup: "UK_VODA",
		},
	}

	container := &CarrierContainer{
		carriers: mockCarriers,
	}

	// Test Success
	source := container.FindCarrierBySource("234", "01")
	assert.Equal(t, "UK_VODA", source)

	// Test Empty MCC
	source = container.FindCarrierBySource("", "01")
	assert.Equal(t, "?", source)

	// Test Carrier Not Found
	source = container.FindCarrierBySource("505", "01")
	assert.Equal(t, "?", source)
}

func TestCarrierContainer_Shutdown(t *testing.T) {
	shutdownCalled := false
	shutdown := func() {
		shutdownCalled = true
	}

	container := &CarrierContainer{
		shutdown: shutdown,
	}

	container.Shutdown()
	assert.True(t, shutdownCalled)

	// Should not panic if shutdown is nil
	nilShutdownContainer := &CarrierContainer{}
	assert.NotPanics(t, func() { nilShutdownContainer.Shutdown() })
}

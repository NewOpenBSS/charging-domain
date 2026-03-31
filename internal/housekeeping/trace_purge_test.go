package housekeeping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPurgeOldTraces(t *testing.T) {
	now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)
	threshold := 36 * time.Hour

	tests := []struct {
		name      string
		tag       pgconn.CommandTag
		execErr   error
		wantCount int64
		wantErr   bool
	}{
		{
			name:      "deletes N rows successfully",
			tag:       pgconn.NewCommandTag("DELETE 12"),
			execErr:   nil,
			wantCount: 12,
			wantErr:   false,
		},
		{
			name:      "zero rows deleted",
			tag:       pgconn.NewCommandTag("DELETE 0"),
			execErr:   nil,
			wantCount: 0,
			wantErr:   false,
		},
		{
			name:      "store error is propagated",
			tag:       pgconn.CommandTag{},
			execErr:   errors.New("timeout"),
			wantCount: 0,
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := &mockDBTX{}
			mockDB.On("Exec", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.tag, tc.execErr)

			svc := newTestService(mockDB)
			count, err := svc.PurgeOldTraces(context.Background(), now, threshold)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "delete old charging trace")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, count)
			}
			mockDB.AssertExpectations(t)
		})
	}
}

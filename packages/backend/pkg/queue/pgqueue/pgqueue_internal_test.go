package pgqueue

// The tick interval is unexported, and it is the one field with real defaulting
// behaviour: time.NewTicker panics on a non-positive duration, so an omitted
// config value must never reach RunTicker.

import (
	"log/slog"
	"testing"
	"time"

	"github.com/NikolayS/pgque-go"
	"github.com/stretchr/testify/require"
)

func TestWithTick(t *testing.T) {
	t.Parallel()

	type args struct {
		ticks []time.Duration // applied in order, so a later option can override
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "success - the default when the option is not passed",
			args: args{},
			want: 500 * time.Millisecond,
		},
		{
			name: "success - a positive interval is kept",
			args: args{ticks: []time.Duration{2 * time.Second}},
			want: 2 * time.Second,
		},
		{
			name: "success - a zero interval keeps the default",
			args: args{ticks: []time.Duration{0}},
			want: 500 * time.Millisecond,
		},
		{
			name: "success - a negative interval keeps the default",
			args: args{ticks: []time.Duration{-time.Second}},
			want: 500 * time.Millisecond,
		},
		{
			name: "success - a zero interval does not undo a configured one",
			args: args{ticks: []time.Duration{2 * time.Second, 0}},
			want: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := []Option{
				WithLogger(slog.New(slog.DiscardHandler)),
				WithQueue("dckz"),
				WithConsumer("dckz-worker"),
				WithClient(&pgque.Client{}),
			}
			for _, tick := range tt.args.ticks {
				opts = append(opts, WithTick(tick))
			}

			q, err := New(opts...)

			require.NoError(t, err)
			require.Equal(t, tt.want, q.tick)
			require.Positive(t, q.tick, "time.NewTicker panics on a non-positive interval")
		})
	}
}

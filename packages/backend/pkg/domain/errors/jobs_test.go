package errors_test

import (
	"errors"
	"fmt"
	"testing"

	errs "dockzilla/pkg/domain/errors"
	"github.com/stretchr/testify/require"
)

func TestIsTerminal(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")

	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success - nil is not terminal",
			args: args{err: nil},
			want: false,
		},
		{
			name: "success - a plain error is retryable",
			args: args{err: errBoom},
			want: false,
		},
		{
			name: "success - a terminal error is terminal",
			args: args{err: errs.Terminal(errBoom)},
			want: true,
		},
		{
			name: "success - terminal survives wrapping",
			args: args{err: fmt.Errorf("decode payload: %w", errs.Terminal(errBoom))},
			want: true,
		},
		{
			name: "success - terminal survives joining",
			args: args{err: errors.Join(errBoom, errs.Terminal(errBoom))},
			want: true,
		},
		{
			name: "success - wrapping a plain error stays retryable",
			args: args{err: fmt.Errorf("send job: %w", errBoom)},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.want, errs.IsTerminal(tt.args.err))
		})
	}
}

func TestTerminal_PreservesTheWrappedError(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	wrapped := errs.Terminal(errBoom)

	require.True(t, errs.IsTerminal(wrapped))
	require.EqualError(t, wrapped, "boom")

	// terminal declares Unwrap, so classifying an error as non-retryable does
	// not cost callers the ability to match the sentinel underneath. The
	// registry depends on this: it marks a missing schema terminal and the
	// HTTP layer still has to recognise it as ErrSchemaNotFound to answer 404.
	require.ErrorIs(t, wrapped, errBoom)
}

func TestSentinels_AreDistinct(t *testing.T) {
	t.Parallel()

	sentinels := []error{
		errs.ErrPayloadTooLarge,
		errs.ErrPayloadEmpty,
		errs.ErrNoTransaction,
		errs.ErrUnsupportedOption,
	}

	for i, outer := range sentinels {
		for j, inner := range sentinels {
			if i == j {
				continue
			}

			require.NotErrorIs(t, outer, inner)
		}
	}
}

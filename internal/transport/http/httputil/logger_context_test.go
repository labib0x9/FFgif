package httputil_test

import (
	"context"
	"testing"

	"github.com/labib0x9/ffgif/internal/transport/http/httputil"
)

func TestLoggerContext(t *testing.T) {
	ctx := context.Background()

	if id := httputil.GetRequestID(ctx); id != "" {
		t.Errorf("expected empty request ID for bare context, got %s", id)
	}

	testID := "test-request-id-999"
	ctxWithID := httputil.WithLoggerContext(ctx, testID)

	if id := httputil.GetRequestID(ctxWithID); id != testID {
		t.Errorf("expected request ID %s, got %s", testID, id)
	}
}

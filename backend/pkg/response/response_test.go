package response

import (
	"net/http"
	"testing"

	"smart-space-management/backend/pkg/errcode"
)

func TestCodeToHTTPMembershipAndOrder(t *testing.T) {
	if got := codeToHTTP(errcode.ErrPhoneRequired); got != http.StatusBadRequest {
		t.Fatalf("codeToHTTP(ErrPhoneRequired)=%d want %d", got, http.StatusBadRequest)
	}
	if got := codeToHTTP(errcode.ErrOrderStatusInvalid); got != http.StatusConflict {
		t.Fatalf("codeToHTTP(ErrOrderStatusInvalid)=%d want %d", got, http.StatusConflict)
	}
}

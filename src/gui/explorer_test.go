package gui

import (
	"net/http"
	"testing"

	"encoding/json"
	"net/http/httptest"
	"strings"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/visor"
	"github.com/skycoin/skycoin/src/visor/historydb"
)

// GetAddressTxns returns a *visor.TransactionResults
func (gw *FakeGateway) GetAddressTxns(a cipher.Address) (*visor.TransactionResults, error) {
	args := gw.Called(a)
	return args.Get(0).(*visor.TransactionResults), args.Error(1)
}

// GetUxOutByID gets UxOut by hash id.
func (gw *FakeGateway) GetUxOutByID(id cipher.SHA256) (*historydb.UxOut, error) {
	args := gw.Called(id)
	return args.Get(0).(*historydb.UxOut), args.Error(1)

}

// GetAddressCount returns count number of unique address with uxouts > 0.
func (gw *FakeGateway) GetAddressCount() (uint64, error) {
	args := gw.Called()
	return args.Get(0).(uint64), args.Error(1)
}

func TestGetAddressCount(t *testing.T) {
	type Result struct {
		Count uint64
	}
	tt := []struct {
		name                         string
		method                       string
		url                          string
		status                       int
		err                          string
		gatewayGetAddressCountResult uint64
		gatewayGetAddressCountErr    error
		result                       Result
	}{
		{
			"405",
			http.MethodPost,
			"/addresscount",
			http.StatusMethodNotAllowed,
			"405 Method Not Allowed",
			0,
			nil,
			Result{},
		},
		{
			"500 - gw GetAddressCount error",
			http.MethodGet,
			"/addresscount",
			http.StatusInternalServerError,
			"500 Internal Server Error",
			0,
			errors.New("gatewayGetAddressCountErr"),
			Result{},
		},
		{
			"200",
			http.MethodGet,
			"/addresscount",
			http.StatusOK,
			"",
			1,
			nil,
			Result{
				Count: 1,
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gateway := &FakeGateway{
				t: t,
			}
			gateway.On("GetAddressCount").Return(tc.gatewayGetAddressCountResult, tc.gatewayGetAddressCountErr)

			req, err := http.NewRequest(tc.method, tc.url, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(getAddressCount(gateway))

			handler.ServeHTTP(rr, req)

			status := rr.Code
			require.Equal(t, tc.status, status, "case: %s, handler returned wrong status code: got `%v` want `%v`", tc.name, status, tc.status)

			if status != http.StatusOK {
				require.Equal(t, tc.err, strings.TrimSpace(rr.Body.String()), "case: %s, handler returned wrong error message: got `%v`| %s, want `%v`",
					tc.name, strings.TrimSpace(rr.Body.String()), status, tc.err)
			} else {
				var msg Result
				err := json.Unmarshal(rr.Body.Bytes(), &msg)
				require.NoError(t, err)
				require.Equal(t, tc.result, msg)
			}
		})
	}
}

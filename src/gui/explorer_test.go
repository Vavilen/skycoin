package gui

import (
	"net/http"
	"testing"

	"encoding/json"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/testutil"
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

// GetRichlist returns rich list as desc order.
func (gw *FakeGateway) GetRichlist(includeDistribution bool) (visor.Richlist, error) {
	args := gw.Called(includeDistribution)
	return args.Get(0).(visor.Richlist), args.Error(1)
}

func TestGetTransactionsForAddress(t *testing.T) {
	address := testutil.MakeAddress()
	successAddress := "111111111111111111111691FSP"
	invalidHash := "caicb"
	validHash := "79216473e8f2c17095c6887cc9edca6c023afedfac2e0c5460e8b6f359684f8b"
	tt := []struct {
		name                        string
		method                      string
		url                         string
		status                      int
		err                         string
		addressParam                string
		gatewayGetAddressTxnsResult *visor.TransactionResults
		gatewayGetAddressTxnsErr    error
		gatewayGetUxOutByIDArg      cipher.SHA256
		gatewayGetUxOutByIDResult   *historydb.UxOut
		gatewayGetUxOutByIDErr      error
		result                      []ReadableTransaction
	}{
		{
			"405",
			http.MethodPost,
			"/explorer/address",
			http.StatusMethodNotAllowed,
			"405 Method Not Allowed",
			"0",
			&visor.TransactionResults{},
			nil,
			cipher.SHA256{},
			nil,
			nil,
			nil,
		},
		{
			"400 - address is empty",
			http.MethodGet,
			"/explorer/address",
			http.StatusBadRequest,
			"400 Bad Request - address is empty",
			"",
			&visor.TransactionResults{},
			nil,
			cipher.SHA256{},
			nil,
			nil,
			nil,
		},
		{
			"400 - invalid address",
			http.MethodGet,
			"/explorer/address",
			http.StatusBadRequest,
			"400 Bad Request - invalid address",
			"badAddress",
			&visor.TransactionResults{},
			nil,
			cipher.SHA256{},
			nil,
			nil,
			nil,
		},
		{
			"500 - gw GetAddressTxns error",
			http.MethodGet,
			"/explorer/address",
			http.StatusInternalServerError,
			"500 Internal Server Error",
			address.String(),
			&visor.TransactionResults{},
			errors.New("gatewayGetAddressTxnsErr"),
			cipher.SHA256{},
			nil,
			nil,
			nil,
		},
		{
			"500 - cipher.SHA256FromHex(tx.Transaction.In) error",
			http.MethodGet,
			"/explorer/address",
			http.StatusInternalServerError,
			"500 Internal Server Error",
			address.String(),
			&visor.TransactionResults{
				Txns: []visor.TransactionResult{
					{
						Transaction: visor.ReadableTransaction{
							In: []string{
								invalidHash,
							},
						},
					},
				},
			},
			nil,
			cipher.SHA256{},
			nil,
			nil,
			nil,
		},
		{
			"500 - GetUxOutByID error",
			http.MethodGet,
			"/explorer/address",
			http.StatusInternalServerError,
			"500 Internal Server Error",
			address.String(),
			&visor.TransactionResults{
				Txns: []visor.TransactionResult{
					{
						Transaction: visor.ReadableTransaction{
							In: []string{
								validHash,
							},
						},
					},
				},
			},
			nil,
			testutil.SHA256FromHex(t, validHash),
			nil,
			errors.New("gatewayGetUxOutByIDErr"),
			nil,
		},
		{
			"500 - GetUxOutByID nil result",
			http.MethodGet,
			"/explorer/address",
			http.StatusInternalServerError,
			"500 Internal Server Error",
			address.String(),
			&visor.TransactionResults{
				Txns: []visor.TransactionResult{
					{
						Transaction: visor.ReadableTransaction{
							In: []string{
								validHash,
							},
						},
					},
				},
			},
			nil,
			testutil.SHA256FromHex(t, validHash),
			nil,
			nil,
			nil,
		},
		{
			"200",
			http.MethodGet,
			"/explorer/address",
			http.StatusOK,
			"",
			address.String(),
			&visor.TransactionResults{
				Txns: []visor.TransactionResult{
					{
						Transaction: visor.ReadableTransaction{
							In: []string{
								validHash,
							},
						},
					},
				},
			},
			nil,
			testutil.SHA256FromHex(t, validHash),
			&historydb.UxOut{},
			nil,
			[]ReadableTransaction{
				{
					In: []visor.ReadableTransactionInput{
						{
							Hash:    validHash,
							Address: successAddress,
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gateway := &FakeGateway{
				t: t,
			}
			gateway.On("GetAddressTxns", address).Return(tc.gatewayGetAddressTxnsResult, tc.gatewayGetAddressTxnsErr)
			gateway.On("GetUxOutByID", tc.gatewayGetUxOutByIDArg).Return(tc.gatewayGetUxOutByIDResult, tc.gatewayGetUxOutByIDErr)

			v := url.Values{}
			var urlFull = tc.url
			if tc.addressParam != "" {
				v.Add("address", tc.addressParam)
			}

			if len(v) > 0 {
				urlFull += "?" + v.Encode()
			}

			req, err := http.NewRequest(tc.method, urlFull, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(getTransactionsForAddress(gateway))

			handler.ServeHTTP(rr, req)

			status := rr.Code
			require.Equal(t, tc.status, status, "case: %s, handler returned wrong status code: got `%v` want `%v`", tc.name, status, tc.status)

			if status != http.StatusOK {
				require.Equal(t, tc.err, strings.TrimSpace(rr.Body.String()), "case: %s, handler returned wrong error message: got `%v`| %s, want `%v`",
					tc.name, strings.TrimSpace(rr.Body.String()), status, tc.err)
			} else {
				var msg []ReadableTransaction
				err := json.Unmarshal(rr.Body.Bytes(), &msg)
				require.NoError(t, err)
				require.Equal(t, tc.result, msg)
			}
		})
	}
}

func TestGetRichlist(t *testing.T) {
	type httpParams struct {
		topn                string
		includeDistribution string
	}
	tt := []struct {
		name                     string
		method                   string
		url                      string
		status                   int
		err                      string
		httpParams               *httpParams
		includeDistribution      bool
		gatewayGetRichlistResult visor.Richlist
		gatewayGetRichlistErr    error
		result                   visor.Richlist
	}{
		{
			"405",
			http.MethodPost,
			"/richlist",
			http.StatusMethodNotAllowed,
			"405 Method Not Allowed",
			nil,
			false,
			visor.Richlist{},
			nil,
			visor.Richlist{},
		},
		{
			"400 - bad topn param",
			http.MethodGet,
			"/richlist",
			http.StatusBadRequest,
			"400 Bad Request - invalid n",
			&httpParams{
				topn: "bad topn",
			},
			false,
			visor.Richlist{},
			nil,
			visor.Richlist{},
		},
		{
			"400 - include-distribution",
			http.MethodGet,
			"/richlist",
			http.StatusBadRequest,
			"400 Bad Request - invalid include-distribution",
			&httpParams{
				topn:                "1",
				includeDistribution: "bad include-distribution",
			},
			false,
			visor.Richlist{},
			nil,
			visor.Richlist{},
		},
		{
			"500 - gw GetRichlist error",
			http.MethodGet,
			"/richlist",
			http.StatusInternalServerError,
			"500 Internal Server Error",
			&httpParams{
				topn:                "1",
				includeDistribution: "false",
			},
			false,
			visor.Richlist{},
			errors.New("gatewayGetRichlistErr"),
			visor.Richlist{},
		},
		{
			"200",
			http.MethodGet,
			"/richlist",
			http.StatusOK,
			"",
			&httpParams{
				topn:                "1",
				includeDistribution: "false",
			},
			false,
			visor.Richlist{},
			nil,
			visor.Richlist{},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			gateway := &FakeGateway{
				t: t,
			}
			gateway.On("GetRichlist", tc.includeDistribution).Return(tc.gatewayGetRichlistResult, tc.gatewayGetRichlistErr)

			v := url.Values{}
			var urlFull = tc.url
			if tc.httpParams != nil {
				if tc.httpParams.topn != "" {
					v.Add("n", tc.httpParams.topn)
				}
				if tc.httpParams.includeDistribution != "" {
					v.Add("include-distribution", tc.httpParams.includeDistribution)
				}
			}
			if len(v) > 0 {
				urlFull += "?" + v.Encode()
			}

			req, err := http.NewRequest(tc.method, urlFull, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(getRichlist(gateway))

			handler.ServeHTTP(rr, req)

			status := rr.Code
			require.Equal(t, tc.status, status, "case: %s, handler returned wrong status code: got `%v` want `%v`", tc.name, status, tc.status)

			if status != http.StatusOK {
				require.Equal(t, tc.err, strings.TrimSpace(rr.Body.String()), "case: %s, handler returned wrong error message: got `%v`| %s, want `%v`",
					tc.name, strings.TrimSpace(rr.Body.String()), status, tc.err)
			} else {
				var msg visor.Richlist
				err := json.Unmarshal(rr.Body.Bytes(), &msg)
				require.NoError(t, err)
				require.Equal(t, tc.result, msg)
			}
		})
	}
}

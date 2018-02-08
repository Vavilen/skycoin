//+build !test

package cmd

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"

	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/gui"
	"github.com/skycoin/skycoin/src/visor"
)

var (
	WebInterface     = true
	webInterfacePort = 6420
	webInterfaceAddr = "127.0.0.1"
)

func getCsrfToken() (string, error) {
	endpoint := fmt.Sprintf("http://%s:%d/csrf", webInterfaceAddr, webInterfacePort)
	resp, err := http.Get(endpoint)
	if err != nil {
		fmt.Printf("failed get /csrf. err: %s\n", err)
		os.Exit(1)
	}
	if resp.Body == nil {
		fmt.Println("failed get /csrf. resp.Body == nil")
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(fmt.Sprintf("failed get blocks. got response status: %d", resp.StatusCode))
	}
	var token struct {
		Token string `json:"csrf_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&token)
	if err != nil {
		fmt.Printf("failed unmarshal resp.Body. err: %s\n", err)
		os.Exit(1)
	}
	return token.Token, err
}

func getBlocks() (*visor.ReadableBlocks, error) {
	endpoint := fmt.Sprintf("http://%s:%d/blocks", webInterfaceAddr, webInterfacePort)
	v := url.Values{}
	v.Add("start", "1")
	v.Add("end", "2")
	if len(v) > 0 {
		endpoint += "?" + v.Encode()
	}
	resp, err := http.Get(endpoint)
	if err != nil {
		fmt.Printf("failed get /blocks. err: %s\n", err)
		os.Exit(1)
	}
	if resp.Body == nil {
		fmt.Println("failed get /blocks. resp.Body == nil")
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("failed get blocks. got response status: %d", resp.StatusCode))
	}
	var blocks *visor.ReadableBlocks
	err = json.NewDecoder(resp.Body).Decode(&blocks)
	if err != nil {
		fmt.Printf("failed unmarshal resp.Body. err: %s\n", err)
		os.Exit(1)
	}
	return blocks, err
}

func TestGetBlock(t *testing.T) {
	//TODO remove this line for manual testing
	t.Skip("skipping test for now")
	randomHash := RandSHA256()
	csrfToken, err := getCsrfToken()
	blocks, err := getBlocks()
	invalidBlockSeq := 1000
	invalidBlockSeqStr := "1000"
	require.NoError(t, err)
	require.Equal(t, len(blocks.Blocks), 2, "got wrong number of blocks: %d. need: %d", len(blocks.Blocks), 2)
	if len(blocks.Blocks) != 2 {
		fmt.Printf("needed 2 blocks. got %d blocks", len(blocks.Blocks))
	}
	client := http.Client{}
	tt := []struct {
		name     string
		method   string
		status   int
		token    string
		err      string
		hash     string
		seqStr   string
		seq      uint64
		response *visor.ReadableBlock
	}{
		{
			name:   "405",
			method: http.MethodPost,
			status: http.StatusMethodNotAllowed,
			err:    "405 Method Not Allowed",
			token:  csrfToken,
		},
		{
			name:   "400 - no seq and hash",
			method: http.MethodGet,
			status: http.StatusBadRequest,
			err:    "400 Bad Request - should specify one filter, hash or seq",
		},
		{
			name:   "400 - seq and hash simultaneously",
			method: http.MethodGet,
			status: http.StatusBadRequest,
			err:    "400 Bad Request - should only specify one filter, hash or seq",
			hash:   "hash",
			seqStr: "seq",
		},
		{
			name:   "400 - hash error: encoding/hex err invalid byte: U+0068 'h'",
			method: http.MethodGet,
			status: http.StatusBadRequest,
			err:    "400 Bad Request - encoding/hex: invalid byte: U+0068 'h'",
			hash:   "hash",
		},
		{
			name:   "400 - hash error: encoding/hex: odd length hex string",
			method: http.MethodGet,
			status: http.StatusBadRequest,
			err:    "400 Bad Request - encoding/hex: odd length hex string",
			hash:   "1234abc",
		},
		{
			name:   "400 - hash error: Invalid hex length",
			method: http.MethodGet,
			status: http.StatusBadRequest,
			err:    "400 Bad Request - Invalid hex length",
			hash:   "1234abcd",
		},
		{
			name:   "404 - block by hash does not exist",
			method: http.MethodGet,
			status: http.StatusNotFound,
			err:    "404 Not Found",
			hash:   randomHash.Hex(),
		},
		{
			name:     "200 - got block by hash",
			method:   http.MethodGet,
			status:   http.StatusOK,
			hash:     blocks.Blocks[0].Head.BlockHash,
			response: &blocks.Blocks[0],
		},
		{
			name:   "400 - seq error: invalid syntax",
			method: http.MethodGet,
			status: http.StatusBadRequest,
			err:    "400 Bad Request - strconv.ParseUint: parsing \"seq\": invalid syntax",
			seqStr: "seq",
		},
		{
			name:   "404 - block by seq does not exist",
			method: http.MethodGet,
			status: http.StatusNotFound,
			err:    "404 Not Found",
			seqStr: invalidBlockSeqStr,
			seq:    uint64(invalidBlockSeq),
		},
		{
			name:     "200 - got block by seq",
			method:   http.MethodGet,
			status:   http.StatusOK,
			seqStr:   "1",
			seq:      1,
			response: &blocks.Blocks[0],
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			endpoint := fmt.Sprintf("http://%s:%d/block", webInterfaceAddr, webInterfacePort)
			v := url.Values{}
			if tc.hash != "" {
				v.Add("hash", tc.hash)
			}
			if tc.seqStr != "" {
				v.Add("seq", tc.seqStr)
			}
			if len(v) > 0 {
				endpoint += "?" + v.Encode()
			}
			req, err := http.NewRequest(tc.method, endpoint, nil)
			if len(tc.token) > 0 {
				req.Header.Add(gui.CSRFHeaderName, tc.token)
			}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("failed get /block. err: %s\n", err)
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				require.Equal(t, tc.err, strings.TrimSpace(string(body)), "case: %s, handler returned wrong error message: got `%v`| %s, want `%v`",
					tc.name, strings.TrimSpace(string(body)), resp.StatusCode, tc.err)
			} else {
				var msg *visor.ReadableBlock
				err := json.Unmarshal(body, &msg)
				require.NoError(t, err)
				require.Equal(t, tc.response, msg, "case: "+tc.name)
			}
		})
	}
	return
}

func RandSHA256() cipher.SHA256 {
	b := make([]byte, 128)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Printf("failed generate random 238 bytes. err: %s", err)
		os.Exit(1)
	}
	return cipher.SumSHA256(b)
}

package cert

import (
	"crypto/tls"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"

	"github.com/skycoin/skycoin/src/util/utc"
)

func TestGenerateCert(t *testing.T) {
	defer func() {
		var err error
		err = os.Remove("keytest.pem")
		require.NoError(t, err)
		err = os.Remove("certtest.pem")
		require.NoError(t, err)
	}()
	err := GenerateCert("certtest.pem", "keytest.pem", "127.0.0.1", "org",
		2048, false, utc.Now(), time.Hour*24)
	assert.Nil(t, err)
	_, err = tls.LoadX509KeyPair("certtest.pem", "keytest.pem")
	assert.Nil(t, err)
}

package historydb

import (
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stretchr/testify/assert"

	"github.com/skycoin/skycoin/src/testutil"
	"github.com/skycoin/skycoin/src/visor/bucket"
)

func TestNewHistoryMeta(t *testing.T) {
	db, td := testutil.PrepareDB(t)
	defer td()

	hm, err := newHistoryMeta(db)
	assert.Nil(t, err)
	err = db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte("history_meta"))
		assert.NotNil(t, bkt)
		return nil
	})
	assert.Nil(t, err)
	v, err := hm.v.Get(parsedHeightKey)
	assert.Nil(t, err)
	assert.Nil(t, v)
}

func TestHistoryMetaGetParsedHeight(t *testing.T) {
	db, td := testutil.PrepareDB(t)
	defer td()

	hm, err := newHistoryMeta(db)
	assert.Nil(t, err)
	v, err := hm.ParsedHeight()
	assert.Nil(t, err)
	assert.Equal(t, int64(-1), v)
	assert.Nil(t, hm.v.Put(parsedHeightKey, bucket.Itob(10)))
	v, err = hm.ParsedHeight()
	assert.Nil(t, err)
	assert.Equal(t, int64(10), v)
}

func TestHistoryMetaSetParsedHeight(t *testing.T) {
	db, td := testutil.PrepareDB(t)
	defer td()

	hm, err := newHistoryMeta(db)
	assert.Nil(t, err)
	assert.Nil(t, hm.SetParsedHeight(0))
	v, err := hm.v.Get(parsedHeightKey)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), bucket.Btoi(v))

	// set 10
	err = hm.SetParsedHeight(10)
	assert.Nil(t, err)
	v, err = hm.v.Get(parsedHeightKey)
	assert.Nil(t, err)
	assert.Equal(t, uint64(10), bucket.Btoi(v))
}

package bucket

import (
	"fmt"
	"math/rand"
	"testing"

	"encoding/json"

	"bytes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skycoin/skycoin/src/testutil"
)

type person struct {
	Name string
	Age  int
}

func TestBktUpdate(t *testing.T) {
	testCases := []struct {
		Init      map[string]person
		UpdateAge map[string]int
	}{
		{
			map[string]person{
				"1": person{"XiaoHei", 10},
				"2": person{"XiaoHuang", 11},
			},
			map[string]int{
				"1": 20,
				"2": 21,
			},
		},
	}

	db, cancel := testutil.PrepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("bkt%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		// init value
		for k, v := range tc.Init {
			d, err := json.Marshal(v)
			assert.Nil(t, err)
			err = bkt.Put([]byte(k), d)
			assert.Nil(t, err)
		}

		// update value
		for k, v := range tc.UpdateAge {
			err := bkt.Update([]byte(k), func(val []byte) ([]byte, error) {
				var p person
				if err := json.NewDecoder(bytes.NewReader(val)).Decode(&p); err != nil {
					return nil, err
				}
				p.Age = v
				d, err := json.Marshal(p)
				if err != nil {
					return nil, err
				}
				return d, nil
			})
			assert.Nil(t, err)
		}

		// check the updated value
		for k, v := range tc.UpdateAge {
			val, err := bkt.Get([]byte(k))
			assert.Nil(t, err)
			var p person
			err = json.NewDecoder(bytes.NewReader(val)).Decode(&p)
			assert.Nil(t, err)
			assert.Equal(t, v, p.Age)
		}
	}
}

func TestReset(t *testing.T) {
	db, cancel := testutil.PrepareDB(t)
	defer cancel()

	bkt, err := New([]byte("tete"), db)
	assert.Nil(t, err)

	assert.Nil(t, bkt.Put([]byte("k1"), []byte("v1")))

	assert.Nil(t, bkt.Put([]byte("k2"), []byte("v2")))
	v, err := bkt.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, []byte("v1"), v)
	v, err = bkt.Get([]byte("k2"))
	assert.Equal(t, []byte("v2"), err)

	assert.Nil(t, bkt.Reset())

	v1, err := bkt.Get([]byte("k1"))
	assert.Nil(t, err)
	if v1 != nil {
		t.Fatal("bucket reset failed")
	}

	v2, err := bkt.Get([]byte("k2"))
	assert.Nil(t, err)
	if v2 != nil {
		t.Fatal("bucket reset failed")
	}

}

func TestDelete(t *testing.T) {
	testCases := []struct {
		Name string
		Init map[string]string
		Del  string
		Err  error
	}{
		{
			"Delete exist",
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"a",
			nil,
		},
		{
			"Delete none exist",
			map[string]string{
				"a": "1",
			},
			"b",
			nil,
		},
	}
	db, cancel := testutil.PrepareDB(t)
	defer cancel()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			bkt, err := New([]byte(fmt.Sprintf("abc%d", rand.Int31n(1024))), db)
			assert.Nil(t, err)
			for k, v := range tc.Init {
				err := bkt.Put([]byte(k), []byte(v))
				assert.Nil(t, err)
			}

			err = bkt.Delete([]byte(tc.Del))
			assert.Equal(t, tc.Err, err)

			// check if this value is deleted
			v, err := bkt.Get([]byte(tc.Del))
			assert.Nil(t, err)
			assert.Nil(t, v)
		})
	}
}

func TestGetAll(t *testing.T) {
	testCases := []struct {
		init map[string]string
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
		},
	}
	db, cancel := testutil.PrepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("abc%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		// init bkt
		for k, v := range tc.init {
			err = bkt.Put([]byte(k), []byte(v))
		}
		assert.Nil(t, err)
		// get all
		vs, err := bkt.GetAll()
		assert.Nil(t, err)
		for k, v := range vs {
			assert.Equal(t, string(v), tc.init[k.(string)])
		}
	}
}

func TestRangeUpdate(t *testing.T) {
	testCases := []struct {
		init map[string]string
		up   map[string]string
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
			map[string]string{
				"a": "10",
				"b": "20",
				"c": "30",
			},
		},
	}
	db, cancel := testutil.PrepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("asd%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		for k, v := range tc.init {
			err = bkt.Put([]byte(k), []byte(v))
			assert.Nil(t, err)
		}

		// range update
		err = bkt.RangeUpdate(func(k, v []byte) ([]byte, error) {
			return []byte(tc.up[string(k)]), nil
		})
		assert.Nil(t, err)

		// check if the value has been updated
		for k, v := range tc.up {
			vv, err := bkt.Get([]byte(k))
			assert.Nil(t, err)
			assert.Equal(t, []byte(v), vv)
		}
	}
}

func TestIsExsit(t *testing.T) {
	testCases := []struct {
		init  map[string]string
		k     string
		exist bool
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"a",
			true,
		},
		{
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"b",
			true,
		},
		{
			map[string]string{
				"a": "1",
				"b": "2",
			},
			"c",
			false,
		},
		{
			map[string]string{},
			"c",
			false,
		},
	}

	db, cancel := testutil.PrepareDB(t)
	defer cancel()

	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("asdf%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)

		// init the bucket
		for k, v := range tc.init {
			err = bkt.Put([]byte(k), []byte(v))
			assert.Nil(t, err)
		}
		isExist, err := bkt.IsExist([]byte(tc.k))
		assert.Nil(t, err)
		assert.Equal(t, tc.exist, isExist)
	}
}

func TestForEach(t *testing.T) {
	testCases := []struct {
		init map[string]string
	}{
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
			},
		},
		{
			map[string]string{},
		},
	}
	db, cancel := testutil.PrepareDB(t)
	defer cancel()
	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("fasd%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		for k, v := range tc.init {
			err = bkt.Put([]byte(k), []byte(v))
			assert.Nil(t, err)
		}

		var count int
		err = bkt.ForEach(func(k, v []byte) error {
			count++
			assert.Equal(t, string(v), tc.init[string(k)])
			return nil
		})
		assert.Nil(t, err)
		assert.Equal(t, len(tc.init), count)
	}
}

func TestLen(t *testing.T) {
	testCases := []struct {
		data map[string]string
		len  int
	}{
		{
			map[string]string{},
			0,
		},
		{
			map[string]string{
				"a": "1",
			},
			1,
		},
		{
			map[string]string{
				"a": "1",
				"b": "2",
				"c": "3",
				"d": "4",
			},
			4,
		},
	}

	db, cl := testutil.PrepareDB(t)
	defer cl()
	for _, tc := range testCases {
		bkt, err := New([]byte(fmt.Sprintf("adsf%d", rand.Int31n(1024))), db)
		assert.Nil(t, err)
		for k, v := range tc.data {
			err = bkt.Put([]byte(k), []byte(v))
			assert.Nil(t, err)
		}
		var l int
		l, err = bkt.Len()
		assert.Nil(t, err)
		assert.Equal(t, tc.len, l)
	}
}

func TestBucketIsEmpty(t *testing.T) {
	db, td := testutil.PrepareDB(t)
	defer td()

	bkt, err := New([]byte("bkt1"), db)
	require.Nil(t, err)
	var isEmpty bool
	isEmpty, err = bkt.IsEmpty()
	require.Nil(t, err)
	require.True(t, isEmpty)

	require.Nil(t, bkt.Put([]byte("k1"), []byte("v1")))

	require.Nil(t, err)
	isEmpty, err = bkt.IsEmpty()
	require.Nil(t, err)
	require.False(t, isEmpty)

	err = bkt.Reset()
	require.Nil(t, err)
	isEmpty, err = bkt.IsEmpty()
	require.Nil(t, err)
	require.True(t, isEmpty)
}

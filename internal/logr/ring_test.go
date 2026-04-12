package logr

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRec(i int) Rec {
	var err error
	if i%2 == 1 {
		err = errors.New("test error")
	}

	return Rec{
		Time:  time.Unix(int64(i), int64(i)),
		Error: err,
	}
}

func requireRecEqual(t *testing.T, expected, actual Rec) {
	t.Helper()

	require.True(t, expected.Time.Equal(actual.Time), "unexpected time: expected %v, got %v", expected.Time, actual.Time)
	if expected.Error == nil || actual.Error == nil {
		require.Equal(t, expected.Error, actual.Error)
		return
	}
	require.EqualError(t, actual.Error, expected.Error.Error())
}

func TestNew(t *testing.T) {
	t.Parallel()

	r := New()

	require.NotNil(t, r)
	assert.Equal(t, -1, r.latest)
	assert.Zero(t, r.count)
}

func TestRingGetLastEmpty(t *testing.T) {
	t.Parallel()

	rec, ok := New().GetLast()
	assert.False(t, ok)

	requireRecEqual(t, Rec{}, rec)
}

func TestRingPutAndGetLast(t *testing.T) {
	t.Parallel()

	r := New()
	first := testRec(1)
	second := testRec(2)

	r.Put(first)
	rec, ok := r.GetLast()
	require.True(t, ok)
	requireRecEqual(t, first, rec)

	r.Put(second)
	rec, ok = r.GetLast()
	require.True(t, ok)
	requireRecEqual(t, second, rec)
}

func TestRingSlicePrev(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, New().Slice())
	})

	t.Run("single", func(t *testing.T) {
		t.Parallel()

		r := New()
		r.Put(testRec(1))

		assert.Nil(t, r.Slice())
	})

	t.Run("multiple", func(t *testing.T) {
		t.Parallel()

		r := New()
		recs := []Rec{testRec(1), testRec(2), testRec(3)}
		for _, rec := range recs {
			r.Put(rec)
		}

		prev := r.Slice()
		require.Len(t, prev, 2)

		requireRecEqual(t, recs[1], prev[0])
		requireRecEqual(t, recs[0], prev[1])
	})
}

func TestRingWrapAround(t *testing.T) {
	t.Parallel()

	r := New()
	recs := make([]Rec, 0, maxStatesToStore+2)
	for i := 0; i < maxStatesToStore+2; i++ {
		rec := testRec(i)
		recs = append(recs, rec)
		r.Put(rec)
	}

	last, ok := r.GetLast()
	require.True(t, ok)
	requireRecEqual(t, recs[len(recs)-1], last)

	prev := r.Slice()
	require.Len(t, prev, maxStatesToStore-1)

	expected := []Rec{
		recs[len(recs)-2],
		recs[len(recs)-3],
		recs[len(recs)-4],
		recs[len(recs)-5],
	}
	for i := range expected {
		requireRecEqual(t, expected[i], prev[i])
	}
}

func TestRingRotatesAfterMaxStatesToStore(t *testing.T) {
	t.Parallel()

	recs := []Rec{
		testRec(0),
		testRec(1),
		testRec(2),
		testRec(3),
		testRec(4),
		testRec(5),
	}

	r := New()
	for _, rec := range recs {
		r.Put(rec)
	}

	last, ok := r.GetLast()
	require.True(t, ok)
	requireRecEqual(t, recs[len(recs)-1], last)

	expected := []Rec{
		recs[maxStatesToStore-1],
		recs[maxStatesToStore-2],
		recs[maxStatesToStore-3],
		recs[maxStatesToStore-4],
	}
	res := r.Slice()
	require.Len(t, res, maxStatesToStore-1)
	for i := range expected {
		requireRecEqual(t, expected[i], res[i])
	}
}

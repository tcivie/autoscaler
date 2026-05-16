package scaler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSelector(t *testing.T) {
	t.Parallel()

	t.Run("empty is catch-all", func(t *testing.T) {
		s, err := ParseSelector("")
		require.NoError(t, err)
		assert.True(t, s.IsCatchAll())
	})

	t.Run("key=value parses", func(t *testing.T) {
		s, err := ParseSelector("platform=linux/arm64")
		require.NoError(t, err)
		assert.Equal(t, "platform", s.Key)
		assert.Equal(t, "linux/arm64", s.Value)
	})

	t.Run("malformed errors", func(t *testing.T) {
		for _, in := range []string{"no-eq", "=x", "x="} {
			_, err := ParseSelector(in)
			assert.Error(t, err, in)
		}
	})
}

func TestRouterBucketSpecificWins(t *testing.T) {
	t.Parallel()

	pools := []Pool{
		{Name: "arm", Group: "arm64", Selector: LabelSelector{"platform", "linux/arm64"}},
		{Name: "amd", Group: "amd64", Selector: LabelSelector{"platform", "linux/amd64"}},
		{Name: "all", Group: "all"}, // catch-all
	}
	r := NewRouter(pools)

	tasks := []Task{
		{Labels: map[string]string{"platform": "linux/arm64"}},
		{Labels: map[string]string{"platform": "linux/arm64"}},
		{Labels: map[string]string{"platform": "linux/amd64"}},
		{Labels: map[string]string{"team": "platform"}}, // no platform -> catch-all
		{Labels: nil},                                   // catch-all
	}
	counts := r.Bucket(tasks)
	assert.Equal(t, []int{2, 1, 2}, counts)
}

func TestRouterWithoutCatchAllDropsUnclaimed(t *testing.T) {
	t.Parallel()

	pools := []Pool{
		{Name: "arm", Group: "arm64", Selector: LabelSelector{"platform", "linux/arm64"}},
	}
	r := NewRouter(pools)
	counts := r.Bucket([]Task{
		{Labels: map[string]string{"platform": "linux/amd64"}},
		{Labels: nil},
	})
	assert.Equal(t, []int{0}, counts)
}

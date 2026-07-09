package doubao

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSeedanceTierPriceMapLatestPricing(t *testing.T) {
	pro, ok := GetSeedanceTierPrice("doubao-seedance-2-0-260128")
	require.True(t, ok)
	require.Equal(t, 46.0, pro["low"])
	require.Equal(t, 28.0, pro["low_video"])
	require.Equal(t, 51.0, pro["high"])
	require.Equal(t, 31.0, pro["high_video"])
	require.Equal(t, 26.0, pro["ultra"])
	require.Equal(t, 16.0, pro["ultra_video"])

	fast, ok := GetSeedanceTierPrice("doubao-seedance-2-0-fast-260128")
	require.True(t, ok)
	require.Equal(t, 37.0, fast["low"])
	require.Equal(t, 22.0, fast["low_video"])
	require.NotContains(t, fast, "high")
	require.NotContains(t, fast, "ultra")

	mini, ok := GetSeedanceTierPrice("doubao-seedance-2-0-mini-260615")
	require.True(t, ok)
	require.Equal(t, 23.0, mini["low"])
	require.Equal(t, 14.0, mini["low_video"])
	require.NotContains(t, mini, "high")
	require.NotContains(t, mini, "ultra")
}

func TestSeedanceVideoInputRatios(t *testing.T) {
	ratio, ok := GetVideoInputRatio("doubao-seedance-2-0-260128", "4K", true)
	require.True(t, ok)
	require.InDelta(t, 16.0/46.0, ratio, 0.000001)

	ratio, ok = GetVideoInputRatio("doubao-seedance-2-0-fast-260128", "720p", true)
	require.True(t, ok)
	require.InDelta(t, 22.0/37.0, ratio, 0.000001)

	ratio, ok = GetVideoInputRatio("doubao-seedance-2-0-fast-260128", "1080p", false)
	require.True(t, ok)
	require.Equal(t, 1.0, ratio)

	ratio, ok = GetVideoInputRatio("doubao-seedance-2-0-mini-260615", "480p", true)
	require.True(t, ok)
	require.InDelta(t, 14.0/23.0, ratio, 0.000001)
}

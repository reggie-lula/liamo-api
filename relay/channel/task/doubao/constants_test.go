package doubao

import (
	"testing"
	"time"

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

	newModel, ok := GetSeedanceTierPrice("doubao-seedance-2-5-260628")
	require.True(t, ok)
	require.Equal(t, 70.0, newModel["low"])
	require.Equal(t, 42.0, newModel["low_video"])
	require.Equal(t, 77.0, newModel["high"])
	require.Equal(t, 46.0, newModel["high_video"])
	require.NotContains(t, newModel, "ultra")
	require.NotContains(t, newModel, "ultra_video")
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

func TestSeedanceLimitedDiscountsUseBeijingDates(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		tierKey   string
		at        time.Time
		want      float64
	}{
		{
			name:      "mini discount starts at Beijing midnight",
			modelName: "doubao-seedance-2-0-mini-260615",
			tierKey:   "low",
			at:        time.Date(2026, time.August, 6, 16, 0, 0, 0, time.UTC),
			want:      23.0 * 0.4,
		},
		{
			name:      "fast discount applies to video input",
			modelName: "doubao-seedance-2-0-fast-260128",
			tierKey:   "low_video",
			at:        time.Date(2026, time.September, 7, 15, 59, 59, 0, time.UTC),
			want:      22.0 * 0.75,
		},
		{
			name:      "mini discount ends after Beijing end date",
			modelName: "doubao-seedance-2-0-mini-260615",
			tierKey:   "low",
			at:        time.Date(2026, time.September, 7, 16, 0, 0, 0, time.UTC),
			want:      23.0,
		},
		{
			name:      "2.5 high tier receives discount",
			modelName: "doubao-seedance-2-5-260628",
			tierKey:   "high",
			at:        time.Date(2026, time.August, 18, 0, 0, 0, 0, seedanceBeijingLocation),
			want:      77.0 * 0.72,
		},
		{
			name:      "2.5 low tier is not discounted",
			modelName: "doubao-seedance-2-5-260628",
			tierKey:   "low",
			at:        time.Date(2026, time.August, 18, 0, 0, 0, 0, seedanceBeijingLocation),
			want:      70.0,
		},
		{
			name:      "2.5 discount ends after Beijing end date",
			modelName: "doubao-seedance-2-5-260628",
			tierKey:   "high_video",
			at:        time.Date(2026, time.September, 17, 16, 0, 0, 0, time.UTC),
			want:      46.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, ok := GetSeedanceTierPriceWithDiscount(tt.modelName, tt.tierKey, tt.at)
			require.True(t, ok)
			require.InDelta(t, tt.want, price, 0.000001)
		})
	}
}

package doubao

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/pkg/billingexpr"

	"github.com/stretchr/testify/assert"
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

func TestSeedanceLimitedDiscountsUseBeijingTimeRange(t *testing.T) {
	tests := []struct {
		name      string
		modelName string
		tierKey   string
		at        time.Time
		want      float64
	}{
		{
			name:      "mini is full price immediately before start",
			modelName: "doubao-seedance-2-0-mini-260615",
			tierKey:   "low",
			at:        time.Date(2026, time.August, 7, 13, 59, 59, 0, seedanceBeijingLocation),
			want:      23.0,
		},
		{
			name:      "mini discount starts at Beijing 14:00",
			modelName: "doubao-seedance-2-0-mini-260615",
			tierKey:   "low",
			at:        time.Date(2026, time.August, 7, 6, 0, 0, 0, time.UTC),
			want:      23.0 * 0.4,
		},
		{
			name:      "fast discount applies through final second",
			modelName: "doubao-seedance-2-0-fast-260128",
			tierKey:   "low_video",
			at:        time.Date(2026, time.October, 7, 13, 59, 59, 0, seedanceBeijingLocation),
			want:      22.0 * 0.75,
		},
		{
			name:      "mini discount ends at Beijing 14:00",
			modelName: "doubao-seedance-2-0-mini-260615",
			tierKey:   "low",
			at:        time.Date(2026, time.October, 7, 14, 0, 0, 0, seedanceBeijingLocation),
			want:      23.0,
		},
		{
			name:      "2.5 high tier discount starts at Beijing 14:00",
			modelName: "doubao-seedance-2-5-260628",
			tierKey:   "high",
			at:        time.Date(2026, time.August, 14, 14, 0, 0, 0, seedanceBeijingLocation),
			want:      77.0 * 0.72,
		},
		{
			name:      "2.5 low tier is not discounted",
			modelName: "doubao-seedance-2-5-260628",
			tierKey:   "low",
			at:        time.Date(2026, time.August, 14, 14, 0, 0, 0, seedanceBeijingLocation),
			want:      70.0,
		},
		{
			name:      "2.5 discount applies through final second",
			modelName: "doubao-seedance-2-5-260628",
			tierKey:   "high_video",
			at:        time.Date(2026, time.October, 17, 13, 59, 59, 0, seedanceBeijingLocation),
			want:      46.0 * 0.72,
		},
		{
			name:      "2.5 discount ends at Beijing 14:00",
			modelName: "doubao-seedance-2-5-260628",
			tierKey:   "high_video",
			at:        time.Date(2026, time.October, 17, 14, 0, 0, 0, seedanceBeijingLocation),
			want:      46.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, ok := GetSeedanceTierPriceWithDiscount(tt.modelName, tt.tierKey, 0, tt.at)
			require.True(t, ok)
			assert.InDelta(t, tt.want, price, 0.000001)
		})
	}
}

func TestSeedanceUserDiscountPriorityAndEligibility(t *testing.T) {
	active := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	lastInstant := time.Date(2027, time.June, 30, 23, 59, 59, 999999999, seedanceBeijingLocation)
	expired := time.Date(2027, time.June, 30, 16, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		model string
		tier  string
		user  int
		at    time.Time
		want  float64
		ok    bool
	}{
		{"user 1 fast overrides general promotion", "doubao-seedance-2-0-fast-260128", "low", 1, active, 18.5, true},
		{"user 2 fast video overrides general promotion", "doubao-seedance-2-0-fast-260128", "low_video", 2, active, 11, true},
		{"user 6 mini does not stack general promotion", "doubao-seedance-2-0-mini-260615", "low", 6, active, 9.2, true},
		{"user 1 mini video does not stack general promotion", "doubao-seedance-2-0-mini-260615", "low_video", 1, active, 5.6, true},
		{"other user retains fast general promotion", "doubao-seedance-2-0-fast-260128", "low", 3, active, 27.75, true},
		{"missing user retains mini general promotion", "doubao-seedance-2-0-mini-260615", "low", 0, active, 9.2, true},
		{"other user pays base price after general promotion", "doubao-seedance-2-0-fast-260128", "low", 3, lastInstant, 37, true},
		{"fast covers entire final Beijing second", "doubao-seedance-2-0-fast-260128", "low", 6, lastInstant.UTC(), 18.5, true},
		{"mini video covers entire final Beijing second", "doubao-seedance-2-0-mini-260615", "low_video", 2, lastInstant, 5.6, true},
		{"fast expires at Beijing midnight", "doubao-seedance-2-0-fast-260128", "low", 1, expired, 37, true},
		{"mini expires at Beijing midnight", "doubao-seedance-2-0-mini-260615", "low", 6, expired, 23, true},
		{"unrelated model keeps its promotion", "doubao-seedance-2-5-260628", "high", 1, active, 55.44, true},
		{"unknown model is not priced", "unknown", "low", 1, active, 0, false},
		{"unsupported tier is not fabricated", "doubao-seedance-2-0-fast-260128", "high", 1, active, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, ok := GetSeedanceTierPriceWithDiscount(tt.model, tt.tier, tt.user, tt.at)
			require.Equal(t, tt.ok, ok)
			assert.InDelta(t, tt.want, price, 0.000001)
		})
	}
}

func TestSeedanceUserDiscountUsageTiers(t *testing.T) {
	at := time.Date(2027, time.June, 30, 12, 0, 0, 0, seedanceBeijingLocation)
	for _, userID := range []int{1, 2, 6} {
		for _, modelName := range []string{"doubao-seedance-2-0-fast-260128", "doubao-seedance-2-0-mini-260615"} {
			for _, resolution := range []string{"480p", "720p", "1080p", "4K"} {
				for _, hasVideo := range []bool{false, true} {
					t.Run(fmt.Sprintf("user_%d/%s/%s/video_%t", userID, modelName, resolution, hasVideo), func(t *testing.T) {
						wantTier, wantPrice := "low", 18.5
						if modelName == "doubao-seedance-2-0-mini-260615" {
							wantPrice = 9.2
						}
						if hasVideo {
							wantTier, wantPrice = "low_video", 11
							if modelName == "doubao-seedance-2-0-mini-260615" {
								wantPrice = 5.6
							}
						}
						tier, price, ok := GetSeedanceTierPriceForUsage(modelName, resolution, hasVideo, userID, at)
						require.True(t, ok)
						assert.Equal(t, wantTier, tier)
						assert.InDelta(t, wantPrice, price, 0.000001)
					})
				}
			}
		}
	}
}

// 异步任务必须用提交时的活动价结算，即使完成时活动已结束。
func TestSeedanceUserDiscountFrozenTaskSettlement(t *testing.T) {
	for _, tt := range []struct {
		model        string
		video        bool
		wantEstimate int
		wantActual   int
	}{
		{"doubao-seedance-2-0-fast-260128", false, 18500, 37000},
		{"doubao-seedance-2-0-fast-260128", true, 11000, 22000},
		{"doubao-seedance-2-0-mini-260615", false, 9200, 18400},
		{"doubao-seedance-2-0-mini-260615", true, 5600, 11200},
	} {
		t.Run(fmt.Sprintf("%s/video_%t", tt.model, tt.video), func(t *testing.T) {
			submittedAt := time.Date(2027, time.June, 30, 23, 59, 59, 0, seedanceBeijingLocation)
			tier, price, ok := GetSeedanceTierPriceForUsage(tt.model, "720p", tt.video, 1, submittedAt)
			require.True(t, ok)
			expression := fmt.Sprintf(`tier(%q, u("tokens") * %s / 1000000)`, tier, strconv.FormatFloat(price, 'f', -1, 64))
			snapshot := &billingexpr.BillingSnapshot{
				ExprString:       expression,
				ExprHash:         billingexpr.ExprHashString(expression),
				GroupRatio:       2,
				QuotaPerUnit:     500000,
				ExprVersion:      1,
				TaskUsageBilling: true,
				EstimatedTier:    tier,
			}
			estimate, err := billingexpr.ComputeTieredQuotaWithRequest(snapshot, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: map[string]any{"tokens": 1000.0}})
			require.NoError(t, err)
			assert.Equal(t, tt.wantEstimate, estimate.ActualQuotaAfterGroup)
			_, expiredPrice, ok := GetSeedanceTierPriceForUsage(tt.model, "720p", tt.video, 1, submittedAt.Add(time.Second))
			require.True(t, ok)
			assert.Greater(t, expiredPrice, price)
			settled, err := billingexpr.ComputeTieredQuotaWithRequest(snapshot, billingexpr.TokenParams{}, billingexpr.RequestInput{Usage: map[string]any{"tokens": 2000.0}})
			require.NoError(t, err)
			assert.Equal(t, tt.wantActual, settled.ActualQuotaAfterGroup)
			assert.Nil(t, settled.Clamp)
		})
	}
}

func TestSeedanceUserDiscountOverridesNewGeneralPromotion(t *testing.T) {
	originalRules := seedanceLimitedDiscountRules
	t.Cleanup(func() { seedanceLimitedDiscountRules = originalRules })
	// 模拟未来新增的通用两折活动，覆盖指定用户活动到期前后。
	seedanceLimitedDiscountRules = []seedanceLimitedDiscountRule{
		{
			modelName:  "doubao-seedance-2-0-fast-260128",
			startAt:    time.Date(2027, time.June, 1, 0, 0, 0, 0, seedanceBeijingLocation),
			endAt:      time.Date(2027, time.August, 1, 0, 0, 0, 0, seedanceBeijingLocation),
			multiplier: 0.2,
		},
		{
			modelName:  "doubao-seedance-2-0-mini-260615",
			startAt:    time.Date(2027, time.June, 1, 0, 0, 0, 0, seedanceBeijingLocation),
			endAt:      time.Date(2027, time.August, 1, 0, 0, 0, 0, seedanceBeijingLocation),
			multiplier: 0.2,
		},
	}
	tests := []struct {
		name  string
		model string
		user  int
		at    time.Time
		want  float64
	}{
		{"fast user 1 keeps five tenths", "doubao-seedance-2-0-fast-260128", 1, time.Date(2027, time.June, 30, 12, 0, 0, 0, seedanceBeijingLocation), 18.5},
		{"fast user 2 keeps five tenths", "doubao-seedance-2-0-fast-260128", 2, time.Date(2027, time.June, 30, 12, 0, 0, 0, seedanceBeijingLocation), 18.5},
		{"mini user 6 keeps four tenths", "doubao-seedance-2-0-mini-260615", 6, time.Date(2027, time.June, 30, 12, 0, 0, 0, seedanceBeijingLocation), 9.2},
		{"fast other user takes general promotion", "doubao-seedance-2-0-fast-260128", 3, time.Date(2027, time.June, 30, 12, 0, 0, 0, seedanceBeijingLocation), 7.4},
		{"fast expired user takes only general promotion", "doubao-seedance-2-0-fast-260128", 1, time.Date(2027, time.July, 1, 0, 0, 0, 0, seedanceBeijingLocation), 7.4},
		{"mini expired user takes only general promotion", "doubao-seedance-2-0-mini-260615", 6, time.Date(2027, time.July, 1, 0, 0, 0, 0, seedanceBeijingLocation), 4.6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, price, ok := GetSeedanceTierPriceForUsage(tt.model, "720p", false, tt.user, tt.at)
			require.True(t, ok)
			assert.InDelta(t, tt.want, price, 0.000001)
		})
	}
}

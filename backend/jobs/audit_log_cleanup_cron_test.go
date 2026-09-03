package jobs

import (
	"testing"
	"time"
)

func TestNextMonthStart(t *testing.T) {
	loc := time.UTC
	cases := []struct {
		name string
		now  time.Time
		hour int
		want time.Time
	}{
		{
			name: "mid month schedules first of next month",
			now:  time.Date(2026, 9, 3, 6, 17, 0, 0, loc),
			hour: 3,
			want: time.Date(2026, 10, 1, 3, 0, 0, 0, loc),
		},
		{
			name: "before 03:00 on the 1st uses this month",
			now:  time.Date(2026, 10, 1, 2, 0, 0, 0, loc),
			hour: 3,
			want: time.Date(2026, 10, 1, 3, 0, 0, 0, loc),
		},
		{
			name: "after 03:00 on the 1st uses next month",
			now:  time.Date(2026, 10, 1, 3, 0, 0, 0, loc),
			hour: 3,
			want: time.Date(2026, 11, 1, 3, 0, 0, 0, loc),
		},
		{
			name: "december wraps to january",
			now:  time.Date(2026, 12, 15, 12, 0, 0, 0, loc),
			hour: 3,
			want: time.Date(2027, 1, 1, 3, 0, 0, 0, loc),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextMonthStart(tc.now, tc.hour)
			if !got.Equal(tc.want) {
				t.Fatalf("nextMonthStart(%s)=%s want %s", tc.now, got, tc.want)
			}
		})
	}
}

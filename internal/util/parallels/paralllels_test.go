package parallels_test

import (
	"reflect"
	"testing"

	"github.com/AmbitiousJun/go-emby2openlist/v2/internal/util/parallels"
)

func TestSliceChunk(t *testing.T) {
	type args struct {
		size int
	}
	tests := []struct {
		name       string
		args       args
		wantRanges []parallels.Range
	}{
		{name: "1", args: args{size: 5}, wantRanges: []parallels.Range{{0, 1}, {1, 2}, {2, 3}, {3, 4}, {4, 5}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRanges := parallels.SliceChunk(tt.args.size); !reflect.DeepEqual(gotRanges, tt.wantRanges) {
				t.Errorf("SliceChunk() = %v, want %v", gotRanges, tt.wantRanges)
			}
		})
	}
}

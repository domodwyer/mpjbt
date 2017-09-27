package dstats

import (
	"bytes"
	"testing"

	"google.golang.org/grpc/benchmark/stats"
)

func TestHistogram_WriteCSV(t *testing.T) {
	hg := NewHistogram(stats.HistogramOptions{
		NumBuckets:     50,
		GrowthFactor:   0.1,
		BaseBucketSize: float64(0),
	})

	for i := 0; i < 1000; i++ {
		hg.Add(int64(i))
	}

	want := "LowerBound,UpperBound,Count,Percent,AccumulativePercent\n" +
		"0.0,1.0,1,0.9,0.9\n" +
		"1.0,1.1,1,0.9,1.9\n" +
		"1.1,1.2,0,0.0,1.9\n" +
		"1.2,1.3,0,0.0,1.9\n" +
		"1.3,1.5,0,0.0,1.9\n" +
		"1.5,1.6,0,0.0,1.9\n" +
		"1.6,1.8,0,0.0,1.9\n" +
		"1.8,1.9,0,0.0,1.9\n" +
		"1.9,2.1,1,0.9,2.8\n" +
		"2.1,2.4,0,0.0,2.8\n" +
		"2.4,2.6,0,0.0,2.8\n" +
		"2.6,2.9,0,0.0,2.8\n" +
		"2.9,3.1,1,0.9,3.7\n" +
		"3.1,3.5,0,0.0,3.7\n" +
		"3.5,3.8,0,0.0,3.7\n" +
		"3.8,4.2,1,0.9,4.7\n" +
		"4.2,4.6,0,0.0,4.7\n" +
		"4.6,5.1,1,0.9,5.6\n" +
		"5.1,5.6,0,0.0,5.6\n" +
		"5.6,6.1,1,0.9,6.5\n" +
		"6.1,6.7,0,0.0,6.5\n" +
		"6.7,7.4,1,0.9,7.5\n" +
		"7.4,8.1,1,0.9,8.4\n" +
		"8.1,9.0,0,0.0,8.4\n" +
		"9.0,9.8,1,0.9,9.3\n" +
		"9.8,10.8,1,0.9,10.3\n" +
		"10.8,11.9,1,0.9,11.2\n" +
		"11.9,13.1,2,1.9,13.1\n" +
		"13.1,14.4,1,0.9,14.0\n" +
		"14.4,15.9,1,0.9,15.0\n" +
		"15.9,17.4,2,1.9,16.8\n" +
		"17.4,19.2,2,1.9,18.7\n" +
		"19.2,21.1,2,1.9,20.6\n" +
		"21.1,23.2,2,1.9,22.4\n" +
		"23.2,25.5,2,1.9,24.3\n" +
		"25.5,28.1,3,2.8,27.1\n" +
		"28.1,30.9,2,1.9,29.0\n" +
		"30.9,34.0,4,3.7,32.7\n" +
		"34.0,37.4,3,2.8,35.5\n" +
		"37.4,41.1,4,3.7,39.3\n" +
		"41.1,45.3,4,3.7,43.0\n" +
		"45.3,49.8,4,3.7,46.7\n" +
		"49.8,54.8,5,4.7,51.4\n" +
		"54.8,60.2,6,5.6,57.0\n" +
		"60.2,66.3,6,5.6,62.6\n" +
		"66.3,72.9,6,5.6,68.2\n" +
		"72.9,80.2,8,7.5,75.7\n" +
		"80.2,88.2,8,7.5,83.2\n" +
		"88.2,97.0,9,8.4,91.6\n" +
		"97.0,inf,9,8.4,100.0\n"

	buf := &bytes.Buffer{}
	if err := hg.WriteCSV(buf); err != nil {
		t.Errorf("unexpected err: %v", err)
	}

	if buf.String() != want {
		t.Errorf("\n\tgot:\n%s\n\n\twant:\n%s", buf.String(), want)
	}
}

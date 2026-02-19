package gopixi

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/gracefulearth/gopixi/internal/buffer"
)

func TestDimensionHeaderSize(t *testing.T) {
	headers := allHeaderVariants(Version)

	for _, header := range headers {
		nameLen := rand.Intn(30)
		name := string(make([]byte, nameLen))
		// Test dimension without axis info
		dim := Dimension{
			Name:     name,
			Size:     rand.Int(),
			TileSize: rand.Int(),
			Type:     ChannelUnknown,
		}
		expectedSize := 2 + nameLen + 2*int(header.OffsetSize) + 4 // +4 for type field
		if dim.HeaderSize(header) != expectedSize {
			t.Errorf("unexpected dimension header size without axis info: got %d, want %d", dim.HeaderSize(header), expectedSize)
		}
		
		// Test dimension with axis info
		dimWithAxis := Dimension{
			Name:     name,
			Size:     rand.Int(),
			TileSize: rand.Int(),
			Type:     ChannelFloat32,
			Minimum:  float32(0.0),
			Step:     float32(1.0),
		}
		expectedSizeWithAxis := 2 + nameLen + 2*int(header.OffsetSize) + 4 + 4 + 4 // +4 for type, +4 for min, +4 for step
		if dimWithAxis.HeaderSize(header) != expectedSizeWithAxis {
			t.Errorf("unexpected dimension header size with axis info: got %d, want %d", dimWithAxis.HeaderSize(header), expectedSizeWithAxis)
		}
	}
}

func TestDimensionWriteRead(t *testing.T) {
	headers := allHeaderVariants(Version)

	cases := []Dimension{
		{Name: "nameone", Size: 40, TileSize: 20, Type: ChannelUnknown},
		{Name: "", Size: 50, TileSize: 5, Type: ChannelUnknown},
		{Name: "amuchlongernamethanusualwithlotsofcharacters", Size: 20000000, TileSize: 1, Type: ChannelUnknown},
		// With axis info
		{Name: "time", Size: 100, TileSize: 10, Type: ChannelFloat64, Minimum: float64(0.0), Step: float64(0.1)},
		{Name: "x", Size: 256, TileSize: 64, Type: ChannelInt32, Minimum: int32(-128), Step: int32(1)},
		{Name: "y", Size: 512, TileSize: 128, Type: ChannelFloat32, Minimum: float32(0.0), Step: float32(0.5)},
	}

	for _, c := range cases {
		for _, h := range headers {
			buf := buffer.NewBuffer(10)
			err := c.Write(buf, h)
			if err != nil {
				t.Fatal("write dimension", err)
			}

			readBuf := buffer.NewBufferFrom(buf.Bytes())
			readDim := Dimension{}
			err = (&readDim).Read(readBuf, h)
			if err != nil {
				t.Fatal("read dimension", err)
			}

			if !reflect.DeepEqual(c, readDim) {
				t.Errorf("expected read dimension to be %v, got %v for header %v", c, readDim, h)
			}
		}
	}
}

func TestDimensionTiles(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		tileSize int
		want     int
	}{
		{"size same as tile size", 10, 10, 1},
		{"small size, small tile", 100, 10, 10},
		{"medium size, medium tile", 500, 50, 10},
		{"large size, large tile", 2000, 100, 20},
		{"zero size", 0, 10, 0},
		{"tile not multiple", 100, 11, 10},
		{"large multiple", 86400, 21600, 4},
		{"half large multiple", 43200, 21600, 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dimension := Dimension{
				Size:     test.size,
				TileSize: test.tileSize,
			}
			got := dimension.Tiles()
			if got != test.want {
				t.Errorf("got %d, want %d", got, test.want)
			}
		})
	}
}

func TestDimensionAxisValue(t *testing.T) {
	tests := []struct {
		name      string
		dimension Dimension
		index     int
		want      any
	}{
		{
			name: "float32 axis",
			dimension: Dimension{
				Name:     "x",
				Size:     100,
				TileSize: 10,
				Type:     ChannelFloat32,
				Minimum:  float32(0.0),
				Step:     float32(0.1),
			},
			index: 0,
			want:  float32(0.0),
		},
		{
			name: "float32 axis index 10",
			dimension: Dimension{
				Name:     "x",
				Size:     100,
				TileSize: 10,
				Type:     ChannelFloat32,
				Minimum:  float32(0.0),
				Step:     float32(0.1),
			},
			index: 10,
			want:  float32(1.0),
		},
		{
			name: "int32 axis",
			dimension: Dimension{
				Name:     "y",
				Size:     256,
				TileSize: 64,
				Type:     ChannelInt32,
				Minimum:  int32(-128),
				Step:     int32(1),
			},
			index: 0,
			want:  int32(-128),
		},
		{
			name: "int32 axis index 128",
			dimension: Dimension{
				Name:     "y",
				Size:     256,
				TileSize: 64,
				Type:     ChannelInt32,
				Minimum:  int32(-128),
				Step:     int32(1),
			},
			index: 128,
			want:  int32(0),
		},
		{
			name: "float64 axis",
			dimension: Dimension{
				Name:     "time",
				Size:     1000,
				TileSize: 100,
				Type:     ChannelFloat64,
				Minimum:  float64(0.0),
				Step:     float64(0.01),
			},
			index: 50,
			want:  float64(0.5),
		},
		{
			name: "no axis info",
			dimension: Dimension{
				Name:     "noaxis",
				Size:     100,
				TileSize: 10,
				Type:     ChannelUnknown,
			},
			index: 10,
			want:  nil,
		},
		{
			name: "uint16 axis",
			dimension: Dimension{
				Name:     "z",
				Size:     1000,
				TileSize: 100,
				Type:     ChannelUint16,
				Minimum:  uint16(100),
				Step:     uint16(2),
			},
			index: 5,
			want:  uint16(110),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.dimension.AxisValue(test.index)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %v (%T), want %v (%T)", got, got, test.want, test.want)
			}
		})
	}
}

func TestDimensionMaximum(t *testing.T) {
	tests := []struct {
		name      string
		dimension Dimension
		want      any
	}{
		{
			name: "float32 axis",
			dimension: Dimension{
				Name:     "x",
				Size:     100,
				TileSize: 10,
				Type:     ChannelFloat32,
				Minimum:  float32(0.0),
				Step:     float32(0.1),
			},
			// Note: float32 precision issues, 99 * 0.1 is not exactly 9.9
			want: float32(99) * float32(0.1),
		},
		{
			name: "int32 axis",
			dimension: Dimension{
				Name:     "y",
				Size:     256,
				TileSize: 64,
				Type:     ChannelInt32,
				Minimum:  int32(-128),
				Step:     int32(1),
			},
			want: int32(127),
		},
		{
			name: "float64 axis",
			dimension: Dimension{
				Name:     "time",
				Size:     1000,
				TileSize: 100,
				Type:     ChannelFloat64,
				Minimum:  float64(0.0),
				Step:     float64(0.01),
			},
			want: float64(9.99),
		},
		{
			name: "no axis info",
			dimension: Dimension{
				Name:     "noaxis",
				Size:     100,
				TileSize: 10,
				Type:     ChannelUnknown,
			},
			want: nil,
		},
		{
			name: "zero size",
			dimension: Dimension{
				Name:     "empty",
				Size:     0,
				TileSize: 10,
				Type:     ChannelFloat32,
				Minimum:  float32(0.0),
				Step:     float32(0.1),
			},
			want: nil,
		},
		{
			name: "uint16 axis",
			dimension: Dimension{
				Name:     "z",
				Size:     1000,
				TileSize: 100,
				Type:     ChannelUint16,
				Minimum:  uint16(100),
				Step:     uint16(2),
			},
			want: uint16(2098),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.dimension.Maximum()
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("got %v (%T), want %v (%T)", got, got, test.want, test.want)
			}
		})
	}
}

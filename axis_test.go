package gopixi

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestAxisValidation(t *testing.T) {
	header := NewHeader(binary.LittleEndian, OffsetSize4)
	
	// Test that axis with type but missing minimum/step returns error
	axis := &Axis{
		Type: ChannelFloat32,
		Unit: "meters",
		// Minimum and Step are nil
	}
	
	buf := new(bytes.Buffer)
	err := axis.Write(buf, header)
	if err == nil {
		t.Error("expected error when writing axis with type but missing minimum/step")
	}
	
	// Test that axis with type and minimum but missing step returns error
	axis2 := &Axis{
		Type:    ChannelFloat32,
		Minimum: float32(0.0),
		Unit:    "meters",
		// Step is nil
	}
	
	buf2 := new(bytes.Buffer)
	err = axis2.Write(buf2, header)
	if err == nil {
		t.Error("expected error when writing axis with type and minimum but missing step")
	}
	
	// Test that axis with all fields works
	axis3 := &Axis{
		Type:    ChannelFloat32,
		Minimum: float32(0.0),
		Step:    float32(1.0),
		Unit:    "meters",
	}
	
	buf3 := new(bytes.Buffer)
	err = axis3.Write(buf3, header)
	if err != nil {
		t.Errorf("unexpected error when writing valid axis: %v", err)
	}
}

func TestAxisNilReceiver(t *testing.T) {
	header := NewHeader(binary.LittleEndian, OffsetSize4)
	
	// Test that nil axis returns 4 for HeaderSize (for the type field)
	var axis *Axis = nil
	size := axis.HeaderSize(header)
	if size != 4 {
		t.Errorf("expected HeaderSize to return 4 for nil axis (type field), got %d", size)
	}
	
	// Test that nil axis Write does nothing
	buf := new(bytes.Buffer)
	err := axis.Write(buf, header)
	if err != nil {
		t.Errorf("unexpected error when writing nil axis: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no bytes written for nil axis, got %d bytes", buf.Len())
	}
	
	// Test that nil axis AxisValue returns nil
	val := axis.AxisValue(10)
	if val != nil {
		t.Errorf("expected AxisValue to return nil for nil axis, got %v", val)
	}
}

func TestAxisHeaderSize(t *testing.T) {
	header := NewHeader(binary.LittleEndian, OffsetSize4)
	
	tests := []struct {
		name     string
		axis     *Axis
		expected int
	}{
		{
			name: "nil axis",
			axis: nil,
			expected: 4, // Always includes 4 bytes for type field
		},
		{
			name: "axis with float32 min and step",
			axis: &Axis{
				Type:    ChannelFloat32,
				Minimum: float32(0.0),
				Step:    float32(1.0),
				Unit:    "seconds",
			},
			expected: 4 + 2 + len("seconds") + 4 + 4, // type + unit + min + step
		},
		{
			name: "axis with int64 min and step",
			axis: &Axis{
				Type:    ChannelInt64,
				Minimum: int64(0),
				Step:    int64(1),
				Unit:    "",
			},
			expected: 4 + 2 + 8 + 8, // type + unit + min + step
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			size := test.axis.HeaderSize(header)
			if size != test.expected {
				t.Errorf("expected HeaderSize %d, got %d", test.expected, size)
			}
		})
	}
}

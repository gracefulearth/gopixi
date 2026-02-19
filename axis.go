package gopixi

import (
	"io"

	"github.com/chenxingqiang/go-floatx"
	"github.com/kshard/float8"
	"github.com/shogo82148/float128"
	"github.com/shogo82148/int128"
	"github.com/x448/float16"
)

// Represents optional axis metadata that describes the units and range of a dimension.
type Axis struct {
	Type    ChannelType // The pixi data type of the axis values. Same as channel data types.
	Minimum any         // The starting value of the axis at dimension index 0. Must match Type if present.
	Step    any         // The increment value as the index increments. Must match Type if present.
	Unit    string      // Optional unit description for the axis values (e.g., "seconds", "meters", "nm").
}

// Returns the size in bytes of the axis metadata as it is laid out and written to disk.
// Returns 0 if the axis is nil.
func (a *Axis) HeaderSize(h Header) int {
	if a == nil {
		return 0
	}
	
	size := 2 + len([]byte(a.Unit)) // unit string
	
	// Add size for optional Minimum value
	if a.Minimum != nil && a.Type != ChannelUnknown {
		size += a.Type.Base().Size()
	}
	
	// Add size for optional Step value
	if a.Step != nil && a.Type != ChannelUnknown {
		size += a.Type.Base().Size()
	}
	
	return size
}

// Writes the binary description of the axis to the given stream.
// If the axis is nil, this method does nothing and returns nil.
func (a *Axis) Write(w io.Writer, h Header) error {
	if a == nil {
		return nil
	}
	
	// Validate that if axis is present, minimum and step must not be nil
	if a.Type != ChannelUnknown && (a.Minimum == nil || a.Step == nil) {
		return ErrFormat("axis with type must have both minimum and step values")
	}
	
	// Write unit string
	err := h.WriteFriendly(w, a.Unit)
	if err != nil {
		return err
	}
	
	// Write Minimum value
	minBytes := make([]byte, a.Type.Base().Size())
	a.Type.Base().PutValue(a.Minimum, h.ByteOrder, minBytes)
	_, err = w.Write(minBytes)
	if err != nil {
		return err
	}
	
	// Write Step value
	stepBytes := make([]byte, a.Type.Base().Size())
	a.Type.Base().PutValue(a.Step, h.ByteOrder, stepBytes)
	_, err = w.Write(stepBytes)
	if err != nil {
		return err
	}
	
	return nil
}

// Reads a description of the axis from the given binary stream.
// The encodedType parameter contains the type with flags indicating presence of Minimum/Step.
func (a *Axis) Read(r io.Reader, h Header, encodedType ChannelType) error {
	// Extract base type
	axisType := encodedType.Base()
	a.Type = axisType
	
	// Read unit string
	unit, err := h.ReadFriendly(r)
	if err != nil {
		return err
	}
	a.Unit = unit
	
	// Read optional Minimum value
	if encodedType.HasMin() && axisType != ChannelUnknown {
		minBytes := make([]byte, axisType.Size())
		_, err = r.Read(minBytes)
		if err != nil {
			return err
		}
		a.Minimum = axisType.Value(minBytes, h.ByteOrder)
	}
	
	// Read optional Step value
	if encodedType.HasMax() && axisType != ChannelUnknown {
		stepBytes := make([]byte, axisType.Size())
		_, err = r.Read(stepBytes)
		if err != nil {
			return err
		}
		a.Step = axisType.Value(stepBytes, h.ByteOrder)
	}
	
	return nil
}

// Returns the axis value at the given dimension index i.
// The value is calculated as: i * step + minimum
// Returns nil if the axis is nil or does not have complete information.
func (a *Axis) AxisValue(i int) any {
	if a == nil || a.Type == ChannelUnknown || a.Minimum == nil || a.Step == nil {
		return nil
	}
	
	// Calculate i * step + minimum based on the type
	switch a.Type.Base() {
	case ChannelInt8:
		min, stp := a.Minimum.(int8), a.Step.(int8)
		return int8(i)*stp + min
	case ChannelUint8:
		min, stp := a.Minimum.(uint8), a.Step.(uint8)
		return uint8(i)*stp + min
	case ChannelInt16:
		min, stp := a.Minimum.(int16), a.Step.(int16)
		return int16(i)*stp + min
	case ChannelUint16:
		min, stp := a.Minimum.(uint16), a.Step.(uint16)
		return uint16(i)*stp + min
	case ChannelInt32:
		min, stp := a.Minimum.(int32), a.Step.(int32)
		return int32(i)*stp + min
	case ChannelUint32:
		min, stp := a.Minimum.(uint32), a.Step.(uint32)
		return uint32(i)*stp + min
	case ChannelInt64:
		min, stp := a.Minimum.(int64), a.Step.(int64)
		return int64(i)*stp + min
	case ChannelUint64:
		min, stp := a.Minimum.(uint64), a.Step.(uint64)
		return uint64(i)*stp + min
	case ChannelFloat8:
		min, stp := float64(a.Minimum.(float8.Float8)), float64(a.Step.(float8.Float8))
		return float8.Float8(float64(i)*stp + min)
	case ChannelFloat16:
		min, stp := a.Minimum.(float16.Float16).Float32(), a.Step.(float16.Float16).Float32()
		return float16.Fromfloat32(float32(i)*stp + min)
	case ChannelFloat32:
		min, stp := a.Minimum.(float32), a.Step.(float32)
		return float32(i)*stp + min
	case ChannelFloat64:
		min, stp := a.Minimum.(float64), a.Step.(float64)
		return float64(i)*stp + min
	case ChannelBool:
		// Boolean axis values don't make sense for linear interpolation
		// Just return the minimum value
		return a.Minimum
	case ChannelInt128:
		min, stp := a.Minimum.(int128.Int128), a.Step.(int128.Int128)
		// i * step + minimum
		// Note: dimension indices are always >= 0, so we can safely use H=0
		i128 := int128.Int128{H: 0, L: uint64(i)}
		istep := stp.Mul(i128)
		return min.Add(istep)
	case ChannelUint128:
		min, stp := a.Minimum.(int128.Uint128), a.Step.(int128.Uint128)
		// i * step + minimum
		i128 := int128.Uint128{H: 0, L: uint64(i)}
		istep := stp.Mul(i128)
		return min.Add(istep)
	case ChannelFloat128:
		min, stp := a.Minimum.(float128.Float128), a.Step.(float128.Float128)
		// i * step + minimum
		fi := float128.FromFloat64(float64(i))
		istep := stp.Mul(fi)
		return min.Add(istep)
	case ChannelBFloat16:
		min, stp := a.Minimum.(floatx.BFloat16), a.Step.(floatx.BFloat16)
		minf, stpf := min.Float32(), stp.Float32()
		return floatx.BF16Fromfloat32(float32(i)*stpf + minf)
	default:
		return nil
	}
}

// Returns the maximum axis value based on the dimension size.
// The maximum is calculated as: (size - 1) * step + minimum
// Returns nil if the axis is nil or does not have complete information.
// Note: Maximum is not serialized to the file as it can be derived from the dimension size.
func (a *Axis) Maximum(size int) any {
	if size <= 0 {
		return nil
	}
	return a.AxisValue(size - 1)
}

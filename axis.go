package gopixi

import (
	"io"
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
	
	// Write optional Minimum value
	if a.Minimum != nil && a.Type != ChannelUnknown {
		minBytes := make([]byte, a.Type.Base().Size())
		a.Type.Base().PutValue(a.Minimum, h.ByteOrder, minBytes)
		_, err = w.Write(minBytes)
		if err != nil {
			return err
		}
	}
	
	// Write optional Step value
	if a.Step != nil && a.Type != ChannelUnknown {
		stepBytes := make([]byte, a.Type.Base().Size())
		a.Type.Base().PutValue(a.Step, h.ByteOrder, stepBytes)
		_, err = w.Write(stepBytes)
		if err != nil {
			return err
		}
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
	return a.Type.AxisValue(i, a.Minimum, a.Step)
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

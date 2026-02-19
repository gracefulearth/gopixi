package gopixi

import (
	"io"
	"strconv"
)

// Represents an axis along which tiled, gridded data is stored in a Pixi file. Data sets can have
// one or more dimensions, but never zero. If a dimension is not tiled, then the TileSize should be
// the same as a the total Size.
type Dimension struct {
	Name     string      // Friendly name to refer to the dimension in the layer.
	Size     int         // The total number of elements in the dimension.
	TileSize int         // The size of the tiles in the dimension. Does not need to be a factor of Size.
	Type     ChannelType // Optional type of the axis values. Same as channel data types.
	Minimum  any         // Optional starting value of the axis at dimension index 0. Must match Type if present.
	Step     any         // Optional increment value as the index increments. Must match Type if present.
}

// Get the size in bytes of this dimension description as it is laid out and written to disk.
func (d Dimension) HeaderSize(h Header) int {
	size := 2 + len([]byte(d.Name)) + 2*int(h.OffsetSize) // base size: name + size + tileSize
	
	// Add 4 bytes for the channel type with flags
	size += 4
	
	// Add size for optional Minimum value
	if d.Minimum != nil && d.Type != ChannelUnknown {
		size += d.Type.Base().Size()
	}
	
	// Add size for optional Step value
	if d.Step != nil && d.Type != ChannelUnknown {
		size += d.Type.Base().Size()
	}
	
	return size
}

// Returns the number of tiles in this dimension.
// The number of tiles is calculated by dividing the size of the dimension by the tile size,
// and then rounding up to the nearest whole number if there are any remaining bytes that do not fit into a full tile.
func (d Dimension) Tiles() int {
	tiles := d.Size / d.TileSize
	if d.Size%d.TileSize != 0 {
		tiles += 1
	}
	return tiles
}

// Writes the binary description of the dimenson to the given stream, according to the specification
// in the Pixi header h.
func (d Dimension) Write(w io.Writer, h Header) error {
	if d.Size <= 0 || d.TileSize <= 0 {
		return ErrFormat("dimension size and tile size must be greater than 0")
	}
	if d.Size < d.TileSize {
		return ErrFormat("dimension tile size cannot be larger than dimension total size")
	}
	
	// write the name, then size and tile size
	err := h.WriteFriendly(w, d.Name)
	if err != nil {
		return err
	}
	err = h.WriteOffset(w, int64(d.Size))
	if err != nil {
		return err
	}
	err = h.WriteOffset(w, int64(d.TileSize))
	if err != nil {
		return err
	}
	
	// Set flags based on presence of Minimum/Step values
	axisType := d.Type.WithMin(d.Minimum != nil).WithMax(d.Step != nil)
	if d.Type == ChannelUnknown {
		axisType = ChannelUnknown
	}
	
	// Write the axis type with flags
	err = h.Write(w, axisType)
	if err != nil {
		return err
	}
	
	// Write optional Minimum value
	if d.Minimum != nil && d.Type != ChannelUnknown {
		minBytes := make([]byte, d.Type.Base().Size())
		d.Type.Base().PutValue(d.Minimum, h.ByteOrder, minBytes)
		_, err = w.Write(minBytes)
		if err != nil {
			return err
		}
	}
	
	// Write optional Step value (using Max flag to indicate Step presence)
	if d.Step != nil && d.Type != ChannelUnknown {
		stepBytes := make([]byte, d.Type.Base().Size())
		d.Type.Base().PutValue(d.Step, h.ByteOrder, stepBytes)
		_, err = w.Write(stepBytes)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Reads a description of the dimension from the given binary stream, according to the specification
// in the Pixi header h.
func (d *Dimension) Read(r io.Reader, h Header) error {
	name, err := h.ReadFriendly(r)
	if err != nil {
		return err
	}
	d.Name = name
	
	size, err := h.ReadOffset(r)
	if err != nil {
		return err
	}
	
	tileSize, err := h.ReadOffset(r)
	if err != nil {
		return err
	}
	
	d.Size = int(size)
	d.TileSize = int(tileSize)
	
	// Read the axis type with flags
	var encodedType ChannelType
	err = h.Read(r, &encodedType)
	if err != nil {
		return err
	}
	
	// Extract base type and flags
	d.Type = encodedType.Base()
	
	// Read optional Minimum value
	if encodedType.HasMin() && d.Type != ChannelUnknown {
		minBytes := make([]byte, d.Type.Size())
		_, err = r.Read(minBytes)
		if err != nil {
			return err
		}
		d.Minimum = d.Type.Value(minBytes, h.ByteOrder)
	} else {
		d.Minimum = nil
	}
	
	// Read optional Step value (using Max flag to indicate Step presence)
	if encodedType.HasMax() && d.Type != ChannelUnknown {
		stepBytes := make([]byte, d.Type.Size())
		_, err = r.Read(stepBytes)
		if err != nil {
			return err
		}
		d.Step = d.Type.Value(stepBytes, h.ByteOrder)
	} else {
		d.Step = nil
	}
	
	return nil
}

func (d Dimension) String() string {
	return d.Name + "(" + strconv.Itoa(d.Size) + " / " + strconv.Itoa(d.TileSize) + ")"
}

// Returns the axis value at the given dimension index i.
// The value is calculated as: i * step + minimum
// Returns nil if the dimension does not have axis information (Type, Minimum, or Step are not set).
func (d Dimension) AxisValue(i int) any {
	if d.Type == ChannelUnknown || d.Minimum == nil || d.Step == nil {
		return nil
	}
	
	// Calculate i * step + minimum based on the type
	return d.Type.AxisValue(i, d.Minimum, d.Step)
}

// Returns the maximum axis value based on the dimension size.
// The maximum is calculated as: (Size - 1) * step + minimum
// Returns nil if the dimension does not have axis information (Type, Minimum, or Step are not set).
// Note: Maximum is not serialized to the file as it can be derived from Size.
func (d Dimension) Maximum() any {
	if d.Size <= 0 {
		return nil
	}
	return d.AxisValue(d.Size - 1)
}

package model

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

type Time struct {
	time.Time
}

func (t Time) ExtensionType() int8 {
	return -1
}

func (t Time) Len() int {
	// Time round towards zero time.
	secPrec := t.Round(time.Second)
	remain := t.Sub(secPrec)
	asSecs := secPrec.Unix()
	if remain == 0 && asSecs > 0 && asSecs <= math.MaxUint32 {
		return 4
	}
	if asSecs < 0 || asSecs >= (1<<34) {
		return 12
	}
	return 8
}

func (t Time) MarshalBinaryTo(bytes []byte) error {
	// Time rounded towards zero.
	secPrec := t.Truncate(time.Second)
	remain := t.Sub(secPrec).Nanoseconds()
	asSecs := secPrec.Unix()
	if remain == 0 && asSecs > 0 && asSecs <= math.MaxUint32 {
		if len(bytes) != 4 {
			return fmt.Errorf("expected length 4, got %d", len(bytes))
		}
		binary.BigEndian.PutUint32(bytes, uint32(asSecs))
		return nil
	}
	if asSecs < 0 || asSecs >= (1<<34) {
		if len(bytes) != 12 {
			return fmt.Errorf("expected length 12, got %d", len(bytes))
		}
		binary.BigEndian.PutUint32(bytes[:4], uint32(remain))
		binary.BigEndian.PutUint64(bytes[4:], uint64(asSecs))
		return nil
	}
	if len(bytes) != 8 {
		return fmt.Errorf("expected length 8, got %d", len(bytes))
	}
	binary.BigEndian.PutUint64(bytes, uint64(asSecs)|(uint64(remain)<<34))
	return nil
}

func (t *Time) UnmarshalBinary(bytes []byte) error {
	switch len(bytes) {
	case 4:
		secs := binary.BigEndian.Uint32(bytes)
		t.Time = time.Unix(int64(secs), 0)
		return nil
	case 8:
		data64 := binary.BigEndian.Uint64(bytes)
		nsecs := int64(data64 >> 34)
		if nsecs > 999999999 {
			// In timestamp 64 and timestamp 96 formats, nanoseconds must not be larger than 999999999.
			return fmt.Errorf("nsecs overflow")
		}
		secs := int64(data64 & 0x3ffffffff)
		t.Time = time.Unix(secs, nsecs)
		return nil
	case 12:
		nsecs := int64(binary.BigEndian.Uint32(bytes[:4]))
		if nsecs > 999999999 {
			// In timestamp 64 and timestamp 96 formats, nanoseconds must not be larger than 999999999.
			return fmt.Errorf("nsecs overflow")
		}
		secs := int64(binary.BigEndian.Uint64(bytes[4:]))
		t.Time = time.Unix(secs, nsecs)
		return nil
	}
	return fmt.Errorf("unknown time format length: %v", len(bytes))
}

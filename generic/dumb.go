package generic

import (
	"encoding/json"
	"strconv"
	"time"
)

type EmbeddedJSON []byte

func (f *EmbeddedJSON) UnmarshalJSON(d []byte) error {
	*f = d
	return nil
}

type (
	Float64String   float64
	Int64String     int64
	Uint64String    int64
	ByteString      byte
	UTCDateString   time.Time
	UnixString      time.Time
	UnixMicroString time.Time
)

func (d Float64String) Value() float64     { return float64(d) }
func (d Int64String) Value() int64         { return int64(d) }
func (d Uint64String) Value() uint64       { return uint64(d) }
func (d ByteString) Value() byte           { return byte(d) }
func (d UTCDateString) Value() time.Time   { return time.Time(d) }
func (d UnixString) Value() time.Time      { return time.Time(d) }
func (d UnixMicroString) Value() time.Time { return time.Time(d) }

func (d UnixString) String() string      { return d.Value().String() }
func (d UnixMicroString) String() string { return d.Value().String() }
func (d UTCDateString) String() string   { return d.Value().String() }

func (f *Float64String) UnmarshalJSON(d []byte) error {
	var str string
	if err := json.Unmarshal(d, &str); err != nil {
		str = string(d)
	}
	n, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	*f = Float64String(n)
	return nil
}

func (i *Int64String) UnmarshalJSON(d []byte) error {
	var str string
	if err := json.Unmarshal(d, &str); err != nil {
		str = string(d)
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return err
	}
	*i = Int64String(n)
	return nil
}

func (i *Uint64String) UnmarshalJSON(d []byte) error {
	var str string
	if err := json.Unmarshal(d, &str); err != nil {
		str = string(d)
	}
	n, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	*i = Uint64String(n)
	return nil
}

func (i *ByteString) UnmarshalJSON(d []byte) error {
	var str string
	if err := json.Unmarshal(d, &str); err != nil {
		str = string(d)
	}
	n, err := strconv.ParseUint(str, 10, 8)
	if err != nil {
		return err
	}
	*i = ByteString(n)
	return nil
}

func (i *UnixString) UnmarshalJSON(d []byte) error {
	var str string
	if err := json.Unmarshal(d, &str); err != nil {
		str = string(d)
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return err
	}
	*i = UnixString(time.Unix(n, 0))
	return nil
}

func (i *UnixMicroString) UnmarshalJSON(d []byte) error {
	var str string
	if err := json.Unmarshal(d, &str); err != nil {
		str = string(d)
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return err
	}

	*i = UnixMicroString(time.Unix(0, n*1000))
	return nil
}

func (i *UTCDateString) UnmarshalJSON(d []byte) error {
	var str string
	if err := json.Unmarshal(d, &str); err != nil {
		return err
	}

	t, err := time.ParseInLocation(
		"2006-01-02 15:04:05.000000",
		str,
		time.UTC,
	)
	if err != nil {
		return err
	}

	*i = UTCDateString(t)
	return nil
}

// Code generated by genji.
// DO NOT EDIT!

package testdata

import (
	"errors"

	"github.com/asdine/genji/record"
	"github.com/asdine/genji/value"
)

// GetField implements the field method of the record.Record interface.
func (b *Basic) GetField(name string) (record.Field, error) {
	switch name {
	case "A":
		return record.NewStringField("A", b.A), nil
	case "B":
		return record.NewIntField("B", b.B), nil
	case "C":
		return record.NewInt32Field("C", b.C), nil
	case "D":
		return record.NewInt32Field("D", b.D), nil
	}

	return record.Field{}, errors.New("unknown field")
}

// Iterate through all the fields one by one and pass each of them to the given function.
// It the given function returns an error, the iteration is interrupted.
func (b *Basic) Iterate(fn func(record.Field) error) error {
	var err error

	err = fn(record.NewStringField("A", b.A))
	if err != nil {
		return err
	}

	err = fn(record.NewIntField("B", b.B))
	if err != nil {
		return err
	}

	err = fn(record.NewInt32Field("C", b.C))
	if err != nil {
		return err
	}

	err = fn(record.NewInt32Field("D", b.D))
	if err != nil {
		return err
	}

	return nil
}

// ScanRecord extracts fields from record and assigns them to the struct fields.
// It implements the record.Scanner interface.
func (b *Basic) ScanRecord(rec record.Record) error {
	return rec.Iterate(func(f record.Field) error {
		var err error

		switch f.Name {
		case "A":
			b.A, err = f.DecodeToString()
		case "B":
			b.B, err = f.DecodeToInt()
		case "C":
			b.C, err = f.DecodeToInt32()
		case "D":
			b.D, err = f.DecodeToInt32()
		}
		return err
	})
}

// Scan extracts fields from src and assigns them to the struct fields.
// It implements the driver.Scanner interface.
func (b *Basic) Scan(src interface{}) error {
	r, ok := src.(record.Record)
	if !ok {
		return errors.New("unable to scan record from src")
	}

	return b.ScanRecord(r)
}

// GetField implements the field method of the record.Record interface.
func (b *basic) GetField(name string) (record.Field, error) {
	switch name {
	case "A":
		return record.NewBytesField("A", b.A), nil
	case "B":
		return record.NewUint16Field("B", b.B), nil
	case "C":
		return record.NewFloat32Field("C", b.C), nil
	case "D":
		return record.NewFloat32Field("D", b.D), nil
	}

	return record.Field{}, errors.New("unknown field")
}

// Iterate through all the fields one by one and pass each of them to the given function.
// It the given function returns an error, the iteration is interrupted.
func (b *basic) Iterate(fn func(record.Field) error) error {
	var err error

	err = fn(record.NewBytesField("A", b.A))
	if err != nil {
		return err
	}

	err = fn(record.NewUint16Field("B", b.B))
	if err != nil {
		return err
	}

	err = fn(record.NewFloat32Field("C", b.C))
	if err != nil {
		return err
	}

	err = fn(record.NewFloat32Field("D", b.D))
	if err != nil {
		return err
	}

	return nil
}

// ScanRecord extracts fields from record and assigns them to the struct fields.
// It implements the record.Scanner interface.
func (b *basic) ScanRecord(rec record.Record) error {
	return rec.Iterate(func(f record.Field) error {
		var err error

		switch f.Name {
		case "A":
			b.A, err = f.DecodeToBytes()
		case "B":
			b.B, err = f.DecodeToUint16()
		case "C":
			b.C, err = f.DecodeToFloat32()
		case "D":
			b.D, err = f.DecodeToFloat32()
		}
		return err
	})
}

// Scan extracts fields from src and assigns them to the struct fields.
// It implements the driver.Scanner interface.
func (b *basic) Scan(src interface{}) error {
	r, ok := src.(record.Record)
	if !ok {
		return errors.New("unable to scan record from src")
	}

	return b.ScanRecord(r)
}

// GetField implements the field method of the record.Record interface.
func (p *Pk) GetField(name string) (record.Field, error) {
	switch name {
	case "A":
		return record.NewStringField("A", p.A), nil
	case "B":
		return record.NewInt64Field("B", p.B), nil
	}

	return record.Field{}, errors.New("unknown field")
}

// Iterate through all the fields one by one and pass each of them to the given function.
// It the given function returns an error, the iteration is interrupted.
func (p *Pk) Iterate(fn func(record.Field) error) error {
	var err error

	err = fn(record.NewStringField("A", p.A))
	if err != nil {
		return err
	}

	err = fn(record.NewInt64Field("B", p.B))
	if err != nil {
		return err
	}

	return nil
}

// ScanRecord extracts fields from record and assigns them to the struct fields.
// It implements the record.Scanner interface.
func (p *Pk) ScanRecord(rec record.Record) error {
	return rec.Iterate(func(f record.Field) error {
		var err error

		switch f.Name {
		case "A":
			p.A, err = f.DecodeToString()
		case "B":
			p.B, err = f.DecodeToInt64()
		}
		return err
	})
}

// Scan extracts fields from src and assigns them to the struct fields.
// It implements the driver.Scanner interface.
func (p *Pk) Scan(src interface{}) error {
	r, ok := src.(record.Record)
	if !ok {
		return errors.New("unable to scan record from src")
	}

	return p.ScanRecord(r)
}

// PrimaryKey returns the primary key. It implements the table.PrimaryKeyer interface.
func (p *Pk) PrimaryKey() ([]byte, error) {
	return value.EncodeInt64(p.B), nil
}

/*
  Copyright (c) 2012-2014 José Carlos Nieto, https://menteslibres.net/xiam

  Permission is hereby granted, free of charge, to any person obtaining
  a copy of this software and associated documentation files (the
  "Software"), to deal in the Software without restriction, including
  without limitation the rights to use, copy, modify, merge, publish,
  distribute, sublicense, and/or sell copies of the Software, and to
  permit persons to whom the Software is furnished to do so, subject to
  the following conditions:

  The above copyright notice and this permission notice shall be
  included in all copies or substantial portions of the Software.

  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
  EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
  MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
  NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
  LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
  OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
  WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package sqlutil

import (
	"database/sql"
	"menteslibres.net/gosexy/to"
	"reflect"
	"strings"
	"upper.io/db"
	"upper.io/db/util"
)

type T struct {
	PrimaryKey  string
	ColumnTypes map[string]reflect.Kind
	util.C
}

type QueryChunks struct {
	Fields     []string
	Limit      string
	Offset     string
	Sort       string
	Conditions string
	Arguments  []interface{}
}

func (self *T) ColumnLike(s string) string {
	for col, _ := range self.ColumnTypes {
		if util.CompareColumnToField(s, col) == true {
			return col
		}
	}
	return s
}

func (self *T) fetchResult(item_t reflect.Type, rows *sql.Rows, columns []string) (reflect.Value, error) {
	var item reflect.Value
	var err error

	expecting := len(columns)

	// Allocating results.
	values := make([]*sql.RawBytes, expecting)
	scanArgs := make([]interface{}, expecting)

	for i := range columns {
		scanArgs[i] = &values[i]
	}

	switch item_t.Kind() {
	case reflect.Map:
		item = reflect.MakeMap(item_t)
	case reflect.Struct:
		item = reflect.New(item_t)
	default:
		return item, db.ErrExpectingMapOrStruct
	}

	err = rows.Scan(scanArgs...)

	if err != nil {
		return item, err
	}

	// Range over row values.
	for i, value := range values {

		if value != nil {
			// Real column name
			column := columns[i]
			// Value as string.
			svalue := string(*value)

			var cv reflect.Value

			if _, ok := self.ColumnTypes[column]; ok == true {
				v, _ := to.Convert(svalue, self.ColumnTypes[column])
				cv = reflect.ValueOf(v)
			} else {
				v, _ := to.Convert(svalue, reflect.String)
				cv = reflect.ValueOf(v)
			}

			switch item_t.Kind() {
			// Destination is a map.
			case reflect.Map:
				if cv.Type() != item_t.Elem() {
					if item_t.Elem().Kind() == reflect.Interface {
						cv, _ = util.StringToType(svalue, cv.Type())
					} else {
						cv, _ = util.StringToType(svalue, item_t.Elem())
					}
				}
				if cv.IsValid() {
					item.SetMapIndex(reflect.ValueOf(column), cv)
				}
			// Destionation is a struct.
			case reflect.Struct:

				index := util.GetStructFieldIndex(item_t, column)

				if index == nil {
					continue
				} else {

					// Destination field.
					destf := item.Elem().FieldByIndex(index)

					if destf.IsValid() {
						if cv.Type() != destf.Type() {
							if destf.Type().Kind() != reflect.Interface {
								cv, _ = util.StringToType(svalue, destf.Type())
							}
						}
						// Copying value.
						if cv.IsValid() {
							destf.Set(cv)
						}
					}

				}
			}
		}
	}

	return item, nil
}

// Returns (lowercased) columns names.
func GetRowColumns(rows *sql.Rows) ([]string, error) {
	// Column names.
	columns, err := rows.Columns()

	if err != nil {
		return nil, err
	}

	// Column names to lower case.
	for i, _ := range columns {
		columns[i] = strings.ToLower(columns[i])
	}

	return columns, nil
}

/*
	Copies *sql.Rows into the slice of maps or structs given by the pointer dst.
*/
func (self *T) FetchRow(dst interface{}, rows *sql.Rows) error {

	dstv := reflect.ValueOf(dst)

	if dstv.IsNil() || dstv.Kind() != reflect.Ptr {
		return db.ErrExpectingPointer
	}

	item_v := dstv.Elem()

	columns, err := GetRowColumns(rows)

	if err != nil {
		return err
	}

	next := rows.Next()

	if next == false {
		if err = rows.Err(); err != nil {
			return err
		}
		return db.ErrNoMoreRows
	}

	item, err := self.fetchResult(item_v.Type(), rows, columns)

	if err != nil {
		return err
	}

	item_v.Set(reflect.Indirect(item))

	return nil
}

/*
	Copies *sql.Rows into the slice of maps or structs given by the pointer dst.
*/
func (self *T) FetchRows(dst interface{}, rows *sql.Rows) error {

	// Destination.
	dstv := reflect.ValueOf(dst)

	if dstv.IsNil() || dstv.Kind() != reflect.Ptr {
		return db.ErrExpectingPointer
	}

	if dstv.Elem().Kind() != reflect.Slice {
		return db.ErrExpectingSlicePointer
	}

	if dstv.Kind() != reflect.Ptr || dstv.Elem().Kind() != reflect.Slice || dstv.IsNil() {
		return db.ErrExpectingSliceMapStruct
	}

	columns, err := GetRowColumns(rows)

	if err != nil {
		return err
	}

	slicev := dstv.Elem()
	item_t := slicev.Type().Elem()

	for rows.Next() {

		item, err := self.fetchResult(item_t, rows, columns)

		if err != nil {
			return err
		}

		slicev = reflect.Append(slicev, reflect.Indirect(item))
	}

	rows.Close()

	dstv.Elem().Set(slicev)

	return nil
}

func (self *T) FieldValues(item interface{}, convertFn func(interface{}) interface{}) ([]string, []interface{}, error) {

	fields := []string{}
	values := []interface{}{}

	item_v := reflect.ValueOf(item)
	item_t := item_v.Type()

	if item_t.Kind() == reflect.Ptr {
		// Single derefence. Just in case user passed a pointer to struct instead of a struct.
		item = item_v.Elem().Interface()
		item_v = reflect.ValueOf(item)
		item_t = item_v.Type()
	}

	switch item_t.Kind() {

	case reflect.Struct:

		nfields := item_v.NumField()

		values = make([]interface{}, 0, nfields)
		fields = make([]string, 0, nfields)

		for i := 0; i < nfields; i++ {

			field := item_t.Field(i)

			if field.PkgPath != "" {
				// Field is unexported.
				continue
			}

			// Field options.
			fieldName, fieldOptions := util.ParseTag(field.Tag.Get("db"))

			// Deprecated "field" tag.
			if deprecatedField := field.Tag.Get("field"); deprecatedField != "" {
				fieldName = deprecatedField
			}

			// Deprecated "omitempty" tag.
			if deprecatedOmitEmpty := field.Tag.Get("omitempty"); deprecatedOmitEmpty != "" {
				fieldOptions["omitempty"] = true
			}

			// Deprecated "inline" tag.
			if deprecatedInline := field.Tag.Get("inline"); deprecatedInline != "" {
				fieldOptions["inline"] = true
			}

			// Processing field name.
			if fieldName == "-" {
				continue
			}

			if fieldName == "" {
				fieldName = self.ColumnLike(field.Name)
			}

			// Processing tag options.
			value := item_v.Field(i).Interface()

			if fieldOptions["omitempty"] == true {
				zero := reflect.Zero(reflect.TypeOf(value)).Interface()
				if value == zero {
					continue
				}
			}

			if fieldOptions["inline"] == true {
				infields, invalues, inerr := self.FieldValues(value, convertFn)
				if inerr != nil {
					return nil, nil, inerr
				}
				fields = append(fields, infields...)
				values = append(values, invalues...)
			} else {
				fields = append(fields, fieldName)
				values = append(values, convertFn(value))
			}

		}
	case reflect.Map:
		nfields := item_v.Len()
		values = make([]interface{}, nfields)
		fields = make([]string, nfields)
		mkeys := item_v.MapKeys()

		for i, key_v := range mkeys {
			valv := item_v.MapIndex(key_v)
			fields[i] = self.ColumnLike(to.String(key_v.Interface()))
			values[i] = convertFn(valv.Interface())
		}

	default:
		return nil, nil, db.ErrExpectingMapOrStruct
	}

	return fields, values, nil
}

func NewQueryChunks() *QueryChunks {
	self := &QueryChunks{}
	return self
}

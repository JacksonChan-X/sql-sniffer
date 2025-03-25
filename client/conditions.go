package client

import (
	"encoding/json"
	"reflect"

	"github.com/iancoleman/strcase"
	"gorm.io/gorm"
)

type (
	ConditionFunc = func(*gorm.DB) *gorm.DB
	Conditions    []ConditionFunc
)

func (w *Conditions) Pagination(page, pagesize int) {
	*w = append(*w, pagination(page, pagesize))
}

func (w *Conditions) And(query any, args ...any) {
	*w = append(*w, func(db *gorm.DB) *gorm.DB {
		return db.Where(query, args...)
	})
}

func (w *Conditions) Joins(query string, on ...any) {
	*w = append(*w, func(db *gorm.DB) *gorm.DB {
		return db.Joins(query, on...)
	})
}

func (w *Conditions) Or(query any, args ...any) {
	*w = append(*w, func(db *gorm.DB) *gorm.DB {
		return db.Or(query, args...)
	})
}

func (w *Conditions) Group(column string) {
	*w = append(*w, func(db *gorm.DB) *gorm.DB {
		return db.Group(column)
	})
}

func (w *Conditions) Order(column interface{}) {
	*w = append(*w, func(db *gorm.DB) *gorm.DB {
		return db.Order(column)
	})
}

func pagination(page, pagesize int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page > 0 {
			db = db.Offset((page - 1) * pagesize)
		}
		if pagesize > 0 {
			db = db.Limit(pagesize)
		}
		return db
	}
}

func AsConditions(req any) Conditions {
	val := reflect.ValueOf(req)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	var conds Conditions
	switch val.Kind() {
	case reflect.Struct:
	NextField:
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if field.Anonymous {
				continue
			}
			if field.PkgPath != "" { // unexported
				continue
			}
			op := " = ?"
			var value any
			switch field.Type.Kind() {
			case reflect.Ptr:
				if val.Field(i).IsNil() {
					continue NextField
				}
				value = val.Field(i).Elem().Interface()
			case reflect.Slice:
				if val.Field(i).IsNil() {
					continue NextField
				}
				value = val.Field(i).Interface()
				op = " IN (?)"
			default:
				isZero := reflect.DeepEqual(reflect.Zero(field.Type).Interface(), val.Field(i).Interface())
				if isZero {
					continue NextField
				}
				value = val.Field(i).Interface()
			}
			if value == nil {
				continue
			}
			fieldname := field.Tag.Get("db")
			if fieldname == "" {
				fieldname = strcase.ToSnake(field.Name)
			}
			conds.And(fieldname+op, value)
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			value := val.MapIndex(key).Interface()
			conds.And(strcase.ToSnake(key.String())+" = ?", value)
		}
	default:
		panic("invalid type")
	}
	return conds
}

var (
	typeJsonRawMessage = reflect.TypeOf(json.RawMessage{})
	typeString         = reflect.TypeOf("")
)

func AsMap(req any) map[string]any {
	val := reflect.ValueOf(req)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()
	m := make(map[string]any)
	switch val.Kind() {
	case reflect.Struct:
	NextField:
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if field.Anonymous {
				continue
			}
			if field.PkgPath != "" { // unexported
				continue
			}
			var value any
			switch field.Type.Kind() {
			case reflect.Ptr:
				if val.Field(i).IsNil() {
					continue NextField
				}
				value = val.Field(i).Elem().Interface()
			case reflect.Slice:
				if val.Field(i).IsNil() {
					continue NextField
				}
				if field.Type == typeJsonRawMessage {
					//Convert为字符串
					value = val.Field(i).Convert(typeString).Interface()
				} else {
					value = val.Field(i).Interface()
				}
			default:
				isZero := reflect.DeepEqual(reflect.Zero(field.Type).Interface(), val.Field(i).Interface())
				if isZero {
					continue NextField
				}
				value = val.Field(i).Interface()
			}
			if value == nil {
				continue
			}
			fieldname := field.Tag.Get("db")
			if fieldname == "" {
				fieldname = strcase.ToSnake(field.Name)
			}
			m[fieldname] = value
		}
	case reflect.Map:
		for _, key := range val.MapKeys() {
			value := val.MapIndex(key).Interface()
			m[key.String()] = value
		}
	default:
		panic("invalid type")
	}
	return m
}

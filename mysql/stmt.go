package mysql

import (
	"encoding/binary"
	"fmt"
	"math"
)

type Statement struct {
	ID         uint32
	SQL        string
	FieldCount uint16
	ParamCount uint16
	Args       []any
}

func (stmt *Statement) BindStmtArgs(nullBitmap, paramTypes, paramValues []byte) (err error) {
	if len(paramTypes)/2 != int(stmt.ParamCount) {
		err = ErrMalformPacket
		return
	}

	pos := 0

	var v []byte
	var n int
	var isNull bool

	for i := 0; i < int(stmt.ParamCount); i++ {
		if nullBitmap[i>>3]&(1<<(uint(i)%8)) > 0 {
			stmt.Args[i] = nil
			continue
		}

		tp := paramTypes[i<<1]
		isUnsigned := (paramTypes[(i<<1)+1] & PARAM_UNSIGNED) > 0

		switch tp {
		case MYSQL_TYPE_NULL:
			stmt.Args[i] = nil
			continue

		case MYSQL_TYPE_TINY:
			if len(paramValues) < (pos + 1) {
				err = ErrMalformPacket
				return
			}

			if isUnsigned {
				stmt.Args[i] = paramValues[pos]
			} else {
				stmt.Args[i] = int8(paramValues[pos])
			}

			pos++
			continue

		case MYSQL_TYPE_SHORT, MYSQL_TYPE_YEAR:
			if len(paramValues) < (pos + 2) {
				err = ErrMalformPacket
				return
			}

			if isUnsigned {
				stmt.Args[i] = binary.LittleEndian.Uint16(paramValues[pos : pos+2])
			} else {
				stmt.Args[i] = int16(binary.LittleEndian.Uint16(paramValues[pos : pos+2]))
			}
			pos += 2
			continue

		case MYSQL_TYPE_INT24, MYSQL_TYPE_LONG:
			if len(paramValues) < (pos + 4) {
				err = ErrMalformPacket
				return
			}

			if isUnsigned {
				stmt.Args[i] = binary.LittleEndian.Uint32(paramValues[pos : pos+4])
			} else {
				stmt.Args[i] = int32(binary.LittleEndian.Uint32(paramValues[pos : pos+4]))
			}
			pos += 4
			continue

		case MYSQL_TYPE_LONGLONG:
			if len(paramValues) < (pos + 8) {
				err = ErrMalformPacket
				return
			}

			if isUnsigned {
				stmt.Args[i] = binary.LittleEndian.Uint64(paramValues[pos : pos+8])
			} else {
				stmt.Args[i] = int64(binary.LittleEndian.Uint64(paramValues[pos : pos+8]))
			}
			pos += 8
			continue

		case MYSQL_TYPE_FLOAT:
			if len(paramValues) < (pos + 4) {
				err = ErrMalformPacket
				return
			}

			stmt.Args[i] = math.Float32frombits(binary.LittleEndian.Uint32(paramValues[pos : pos+4]))
			pos += 4
			continue

		case MYSQL_TYPE_DOUBLE:
			if len(paramValues) < (pos + 8) {
				err = ErrMalformPacket
				return
			}

			stmt.Args[i] = math.Float64frombits(binary.LittleEndian.Uint64(paramValues[pos : pos+8]))
			pos += 8
			continue

		case MYSQL_TYPE_DECIMAL, MYSQL_TYPE_NEWDECIMAL, MYSQL_TYPE_VARCHAR,
			MYSQL_TYPE_BIT, MYSQL_TYPE_ENUM, MYSQL_TYPE_SET, MYSQL_TYPE_TINY_BLOB,
			MYSQL_TYPE_MEDIUM_BLOB, MYSQL_TYPE_LONG_BLOB, MYSQL_TYPE_BLOB,
			MYSQL_TYPE_VAR_STRING, MYSQL_TYPE_STRING, MYSQL_TYPE_GEOMETRY,
			MYSQL_TYPE_DATE, MYSQL_TYPE_NEWDATE,
			MYSQL_TYPE_TIMESTAMP, MYSQL_TYPE_DATETIME, MYSQL_TYPE_TIME:
			if len(paramValues) < (pos + 1) {
				err = ErrMalformPacket
				return
			}

			v, isNull, n, err = LengthEncodedString(paramValues[pos:])
			pos += n
			if err != nil {
				return
			}

			if !isNull {
				stmt.Args[i] = v
				continue
			} else {
				stmt.Args[i] = nil
				continue
			}
		default:
			err = fmt.Errorf("STMT UNKNOWN FieldType %d", tp)
		}
	}
	return
}

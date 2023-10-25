package repo

import (
	"reflect"
	"time"
)

type ModelResp interface {
	Fields() []string
}

func ListResp[M TableNameType, MR ModelResp]() ([]*MR, error) {
	var mr MR
	fields := mr.Fields()

	ms, err := Find[M](fields, []KVPair{})
	if err != nil {
		return nil, err
	}

	var list []*MR

	for _, m := range ms {
		list = append(list, m2mr[MR](m, fields))
	}

	return list, nil
}

func ItemResp[M TableNameType, MR ModelResp](m *M) *MR {
	var mr MR
	return m2mr[MR](m, mr.Fields())
}

// 把 m 的字段值，赋值给 mr
// 同时，把 created_at 和 updated_at 字段，自动转换为
// repo.Timestamp 类型
func m2mr[MR any](m any, fields []string) *MR {
	fields = append(fields, "CreatedAt", "UpdatedAt")

	var mr MR

	vm := reflect.ValueOf(m).Elem()
	vmr := reflect.ValueOf(&mr).Elem()

	for _, field := range fields {
		camelName := ToCamel(field)

		var mf = vm.FieldByName(camelName)
		var mrf = vmr.FieldByName(camelName)

		if mrf.CanSet() {
			val := reflect.ValueOf(mf.Interface())

			// try test if the field is time.Time type
			if t, ok := val.Interface().(time.Time); ok {
				val2 := Timestamp(t)
				mrf.Set(reflect.ValueOf(val2))
			} else {
				mrf.Set(val)
			}
		}
	}

	return &mr
}

package repo

import (
	"reflect"
	"time"

	"github.com/daqing/airway/lib/utils"
)

type ModelResp interface {
	Fields() []string
}

func ListResp[M TableNameType, MR ModelResp]() ([]*MR, error) {
	var mr MR
	fields := mr.Fields()

	ms, err := Find[M](fields, []KeyValueField{})
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
// utils.Timestamp 类型
func m2mr[MR any](m any, fields []string) *MR {
	fields = append(fields, "CreatedAt", "UpdatedAt")

	var mr MR

	vm := reflect.ValueOf(m).Elem()
	vmr := reflect.ValueOf(&mr).Elem()

	for _, field := range fields {
		camelName := utils.ToCamel(field)

		var mf = vm.FieldByName(camelName)
		var mrf = vmr.FieldByName(camelName)

		if mrf.CanSet() {
			if camelName == "CreatedAt" || camelName == "UpdatedAt" {
				val := reflect.ValueOf(mf.Interface())

				val2 := utils.Timestamp(val.Interface().(time.Time))

				mrf.Set(reflect.ValueOf(val2))
			} else {
				mrf.Set(reflect.ValueOf(mf.Interface()))
			}

		}
	}

	return &mr
}

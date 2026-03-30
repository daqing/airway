package repo

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	buildingsql "github.com/daqing/airway/lib/sql"
)

// Preloader 用于预加载关联数据，解决N+1问题
type Preloader struct {
	db        *DB
	relations []preloadRelation
}

type preloadRelation struct {
	name       string
	conditions buildingsql.CondBuilder
}

// Preload 开始一个预加载查询链
func (db *DB) Preload(relations ...string) *Preloader {
	p := &Preloader{db: db}
	for _, rel := range relations {
		p.relations = append(p.relations, preloadRelation{name: rel})
	}
	return p
}

// PreloadCond 带条件的预加载
func (db *DB) PreloadCond(relation string, cond buildingsql.CondBuilder) *Preloader {
	return &Preloader{
		db: db,
		relations: []preloadRelation{
			{name: relation, conditions: cond},
		},
	}
}

// ThenPreload 链式添加预加载
func (p *Preloader) ThenPreload(relations ...string) *Preloader {
	for _, rel := range relations {
		p.relations = append(p.relations, preloadRelation{name: rel})
	}
	return p
}

// ThenPreloadCond 链式添加带条件的预加载
func (p *Preloader) ThenPreloadCond(relation string, cond buildingsql.CondBuilder) *Preloader {
	p.relations = append(p.relations, preloadRelation{name: relation, conditions: cond})
	return p
}

// Exec 执行预加载查询
// 传入已查询的主记录，自动加载所有关联数据
func (p *Preloader) Exec(primaryRecords any) error {
	if len(p.relations) == 0 {
		return nil
	}

	v := reflect.ValueOf(primaryRecords)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("primaryRecords must be a pointer to slice")
	}

	sliceVal := v.Elem()
	if sliceVal.Len() == 0 {
		return nil
	}

	// 提取主键值
	pks := extractPrimaryKeys(sliceVal)
	if len(pks) == 0 {
		return nil
	}

	// 逐个加载关联
	for _, rel := range p.relations {
		if err := p.loadRelation(rel, sliceVal, pks); err != nil {
			return err
		}
	}

	return nil
}

// loadRelation 加载单个关联
func (p *Preloader) loadRelation(rel preloadRelation, primarySlice reflect.Value, pks []int64) error {
	// 获取关联模型类型
	elemType := primarySlice.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	relConfig, err := p.resolveRelationConfig(elemType, rel.name)
	if err != nil {
		return err
	}

	// 查询关联数据
	relatedRecords, err := p.fetchRelatedRecords(relConfig, pks, rel.conditions)
	if err != nil {
		return err
	}

	// 将关联数据填充到主记录中
	return p.assignRelatedRecords(primarySlice, relatedRecords, relConfig)
}

// preloadRelConfig 预加载的关联配置
type preloadRelConfig struct {
	name         string
	modelType    reflect.Type
	relationType RelationType
	foreignKey   string // 关联表中的外键字段（Go字段名）
	primaryKey   string // 主表中的主键字段（Go字段名）
	setterField  string // 主模型中用于存储关联数据的字段名
}

// resolveRelationConfig 解析关联配置
func (p *Preloader) resolveRelationConfig(primaryType reflect.Type, relationName string) (*preloadRelConfig, error) {
	// 首先尝试从Relations()方法获取
	rel, err := getRelationConfig(primaryType, relationName)
	if err == nil && rel != nil {
		relType := getModelType(rel.Model)
		return &preloadRelConfig{
			name:         relationName,
			modelType:    relType,
			relationType: rel.Type,
			foreignKey:   rel.ForeignKey,
			primaryKey:   rel.PrimaryKey,
			setterField:  relationName,
		}, nil
	}

	// 尝试从结构体字段推断
	return p.inferRelationConfig(primaryType, relationName)
}

// inferRelationConfig 推断关联配置
func (p *Preloader) inferRelationConfig(primaryType reflect.Type, relationName string) (*preloadRelConfig, error) {
	field, ok := primaryType.FieldByName(relationName)
	if !ok {
		return nil, fmt.Errorf("relation '%s' not found in model '%s'", relationName, primaryType.Name())
	}

	var relType reflect.Type
	var relTypeStr RelationType

	switch field.Type.Kind() {
	case reflect.Slice:
		relType = field.Type.Elem()
		if relType.Kind() == reflect.Ptr {
			relType = relType.Elem()
		}
		relTypeStr = HasMany
	case reflect.Ptr:
		relType = field.Type.Elem()
		relTypeStr = HasOne
	default:
		// 可能是BelongsTo，值类型
		relType = field.Type
		relTypeStr = BelongsTo
	}

	// 推断外键和主键
	fk := primaryType.Name() + "ID"
	pk := "ID"

	if relTypeStr == BelongsTo {
		// BelongsTo: 主表有外键
		fk = relationName + "ID"
	}

	return &preloadRelConfig{
		name:         relationName,
		modelType:    relType,
		relationType: relTypeStr,
		foreignKey:   fk,
		primaryKey:   pk,
		setterField:  relationName,
	}, nil
}

// fetchRelatedRecords 获取关联记录
func (p *Preloader) fetchRelatedRecords(config *preloadRelConfig, pks []int64, cond buildingsql.CondBuilder) (map[int64][]reflect.Value, error) {
	table := getTableNameFromType(config.modelType)

	// 确定查询条件字段
	var dbFK string
	if config.relationType == BelongsTo {
		// BelongsTo: 关联表的主键在主表的外键中
		dbFK = toDBFieldName(config.primaryKey)
	} else {
		// HasOne/HasMany: 关联表的外键等于主表的主键
		dbFK = toDBFieldName(config.foreignKey)
	}

	builder := buildingsql.Select("*").From(table).
		Where(buildingsql.In(dbFK, pks))

	if cond != nil {
		builder = builder.Where(cond)
	}

	query, args := builder.ToSQL()
	compiledQuery, compiledArgs, err := compileNamedQuery(query, args)
	if err != nil {
		return nil, err
	}

	rows, err := p.db.conn.QueryxContext(context.Background(), p.db.conn.Rebind(compiledQuery), compiledArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]reflect.Value)

	for rows.Next() {
		record := reflect.New(config.modelType)
		if err := rows.StructScan(record.Interface()); err != nil {
			return nil, err
		}

		var fk int64
		if config.relationType == BelongsTo {
			// BelongsTo: 使用关联表的主键
			fk = getPKValueFromRecord(record, config.primaryKey)
		} else {
			// HasOne/HasMany: 使用关联表的外键
			fk = getFKValueFromRecord(record, config.foreignKey)
		}
		result[fk] = append(result[fk], record)
	}

	return result, rows.Err()
}

// assignRelatedRecords 将关联记录赋值给主记录
func (p *Preloader) assignRelatedRecords(primarySlice reflect.Value, relatedMap map[int64][]reflect.Value, config *preloadRelConfig) error {
	for i := 0; i < primarySlice.Len(); i++ {
		elem := primarySlice.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		var pk int64
		if config.relationType == BelongsTo {
			// BelongsTo: 使用主表的外键查找关联记录
			pk = getFKValueFromElem(elem, config.foreignKey)
		} else {
			// HasOne/HasMany: 使用主表的主键
			pk = getPKValueFromElem(elem, config.primaryKey)
		}

		relatedRecords := relatedMap[pk]

		// 设置关联字段
		field := elem.FieldByName(config.setterField)
		if !field.IsValid() || !field.CanSet() {
			continue // 字段不存在或不可设置则跳过
		}

		switch config.relationType {
		case HasMany:
			// 设置为切片
			sliceType := reflect.SliceOf(reflect.PtrTo(config.modelType))
			slice := reflect.MakeSlice(sliceType, len(relatedRecords), len(relatedRecords))
			for j, rec := range relatedRecords {
				slice.Index(j).Set(rec)
			}
			field.Set(slice)

		case HasOne:
			// 设置为指针
			if len(relatedRecords) > 0 {
				field.Set(relatedRecords[0])
			}

		case BelongsTo:
			// 设置为指针
			if len(relatedRecords) > 0 {
				field.Set(relatedRecords[0])
			}
		}
	}

	return nil
}

// 辅助函数

func extractPrimaryKeys(slice reflect.Value) []int64 {
	pks := make([]int64, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		elem := slice.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		pk := getPKValueFromElem(elem, "ID")
		if pk != 0 {
			pks = append(pks, pk)
		}
	}
	return pks
}

func getPKValueFromElem(elem reflect.Value, pkField string) int64 {
	field := elem.FieldByName(pkField)
	if !field.IsValid() {
		return 0
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(field.Uint())
	}
	return 0
}

func getFKValueFromElem(elem reflect.Value, fkField string) int64 {
	field := elem.FieldByName(fkField)
	if !field.IsValid() {
		return 0
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(field.Uint())
	}
	return 0
}

func getPKValueFromRecord(record reflect.Value, pkField string) int64 {
	elem := record.Elem()
	return getPKValueFromElem(elem, pkField)
}

func getFKValueFromRecord(record reflect.Value, fkField string) int64 {
	elem := record.Elem()
	return getFKValueFromElem(elem, fkField)
}

func getTableNameFromType(t reflect.Type) string {
	if table, ok := reflect.New(t).Interface().(interface{ TableName() string }); ok {
		return table.TableName()
	}
	return strings.ToLower(t.Name()) + "s"
}

func toDBFieldName(goField string) string {
	var result strings.Builder
	for i, r := range goField {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteByte('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// ==================== 便捷函数（使用CurrentDB）====================

// Preload 开始一个预加载查询链（使用CurrentDB）
func Preload(relations ...string) *Preloader {
	return CurrentDB().Preload(relations...)
}

// PreloadCond 带条件的预加载（使用CurrentDB）
func PreloadCond(relation string, cond buildingsql.CondBuilder) *Preloader {
	return CurrentDB().PreloadCond(relation, cond)
}

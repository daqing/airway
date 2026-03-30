package repo

import (
	"context"
	"fmt"
	"reflect"

	buildingsql "github.com/daqing/airway/lib/sql"
)

// JoinQuery 构建JOIN查询
type JoinQuery struct {
	db         *DB
	modelType  reflect.Type
	joins      []joinDef
	conditions buildingsql.CondBuilder
	orderBys   []string
	limit      int
	offset     int
}

type joinDef struct {
	relation   string
	joinType   string // INNER, LEFT, RIGHT
	alias      string
	conditions buildingsql.CondBuilder
}

// Join 开始一个JOIN查询
func (db *DB) Join(model any) *JoinQuery {
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	return &JoinQuery{
		db:        db,
		modelType: modelType,
		joins:     []joinDef{},
		limit:     -1,
		offset:    -1,
	}
}

// Joins 添加INNER JOIN
func (jq *JoinQuery) Joins(relation string, cond ...buildingsql.CondBuilder) *JoinQuery {
	def := joinDef{relation: relation, joinType: "JOIN"}
	if len(cond) > 0 {
		def.conditions = cond[0]
	}
	jq.joins = append(jq.joins, def)
	return jq
}

// LeftJoins 添加LEFT JOIN
func (jq *JoinQuery) LeftJoins(relation string, cond ...buildingsql.CondBuilder) *JoinQuery {
	def := joinDef{relation: relation, joinType: "LEFT JOIN"}
	if len(cond) > 0 {
		def.conditions = cond[0]
	}
	jq.joins = append(jq.joins, def)
	return jq
}

// RightJoins 添加RIGHT JOIN
func (jq *JoinQuery) RightJoins(relation string, cond ...buildingsql.CondBuilder) *JoinQuery {
	def := joinDef{relation: relation, joinType: "RIGHT JOIN"}
	if len(cond) > 0 {
		def.conditions = cond[0]
	}
	jq.joins = append(jq.joins, def)
	return jq
}

// FullJoins 添加FULL JOIN
func (jq *JoinQuery) FullJoins(relation string, cond ...buildingsql.CondBuilder) *JoinQuery {
	def := joinDef{relation: relation, joinType: "FULL JOIN"}
	if len(cond) > 0 {
		def.conditions = cond[0]
	}
	jq.joins = append(jq.joins, def)
	return jq
}

// CrossJoins 添加CROSS JOIN
func (jq *JoinQuery) CrossJoins(relation string) *JoinQuery {
	jq.joins = append(jq.joins, joinDef{relation: relation, joinType: "CROSS JOIN"})
	return jq
}

// Where 添加查询条件
func (jq *JoinQuery) Where(cond buildingsql.CondBuilder) *JoinQuery {
	jq.conditions = cond
	return jq
}

// OrderBy 添加排序
func (jq *JoinQuery) OrderBy(order string) *JoinQuery {
	jq.orderBys = append(jq.orderBys, order)
	return jq
}

// Limit 设置限制
func (jq *JoinQuery) Limit(limit int) *JoinQuery {
	jq.limit = limit
	return jq
}

// Offset 设置偏移
func (jq *JoinQuery) Offset(offset int) *JoinQuery {
	jq.offset = offset
	return jq
}

// Page 分页
func (jq *JoinQuery) Page(page, perPage int) *JoinQuery {
	if page < 1 {
		page = 1
	}
	jq.limit = perPage
	jq.offset = (page - 1) * perPage
	return jq
}

// buildQuery 构建查询
func (jq *JoinQuery) buildQuery() (*buildingsql.Builder, error) {
	table := getTableNameFromType(jq.modelType)
	builder := buildingsql.Select("*").From(table)

	for _, join := range jq.joins {
		relConfig, err := jq.resolveJoinRelationConfig(jq.modelType, join.relation)
		if err != nil {
			return nil, err
		}

		joinTable := getTableNameFromType(relConfig.modelType)
		onCond := jq.buildJoinCondition(jq.modelType, relConfig)

		if join.conditions != nil {
			onCond = buildingsql.AllOf(onCond, join.conditions)
		}

		switch join.joinType {
		case "LEFT JOIN":
			builder = builder.LeftJoinTable(buildingsql.Ref(joinTable), onCond)
		case "RIGHT JOIN":
			builder = builder.RightJoinTable(buildingsql.Ref(joinTable), onCond)
		case "FULL JOIN":
			builder = builder.FullJoinTable(buildingsql.Ref(joinTable), onCond)
		case "CROSS JOIN":
			builder = builder.CrossJoinTable(buildingsql.Ref(joinTable))
		default:
			builder = builder.JoinTable(buildingsql.Ref(joinTable), onCond)
		}
	}

	if jq.conditions != nil {
		builder = builder.Where(jq.conditions)
	}

	for _, order := range jq.orderBys {
		builder = builder.OrderBy(order)
	}

	if jq.limit >= 0 {
		builder = builder.Limit(jq.limit)
	}

	if jq.offset >= 0 {
		builder = builder.Offset(jq.offset)
	}

	return builder, nil
}

// Find 执行查询并返回结果
func (jq *JoinQuery) Find() (*JoinResultSet, error) {
	builder, err := jq.buildQuery()
	if err != nil {
		return nil, err
	}

	query, args := builder.ToSQL()
	compiledQuery, compiledArgs, err := compileNamedQuery(query, args)
	if err != nil {
		return nil, err
	}

	rows, err := jq.db.conn.QueryxContext(context.Background(), jq.db.conn.Rebind(compiledQuery), compiledArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 收集结果
	var records []map[string]any
	for rows.Next() {
		record := make(map[string]any)
		if err := rows.MapScan(record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &JoinResultSet{
		records:   records,
		modelType: jq.modelType,
		joins:     jq.joins,
	}, nil
}

// FindInto 执行查询并扫描到目标切片
func (jq *JoinQuery) FindInto(dest any) error {
	builder, err := jq.buildQuery()
	if err != nil {
		return err
	}

	query, args := builder.ToSQL()
	compiledQuery, compiledArgs, err := compileNamedQuery(query, args)
	if err != nil {
		return err
	}

	return jq.db.conn.SelectContext(context.Background(), dest, jq.db.conn.Rebind(compiledQuery), compiledArgs...)
}

// Count 返回计数
func (jq *JoinQuery) Count() (int64, error) {
	builder, err := jq.buildQuery()
	if err != nil {
		return 0, err
	}

	// 修改查询为COUNT
	query, args := builder.ToSQL()
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS t", query)

	compiledQuery, compiledArgs, err := compileNamedQuery(countQuery, args)
	if err != nil {
		return 0, err
	}

	var count int64
	err = jq.db.conn.GetContext(context.Background(), &count, jq.db.conn.Rebind(compiledQuery), compiledArgs...)
	return count, err
}

// joinRelationConfig JOIN关联配置
type joinRelationConfig struct {
	name         string
	modelType    reflect.Type
	relationType RelationType
	foreignKey   string // 关联表中的外键字段（Go字段名）
	primaryKey   string // 主表中的主键字段（Go字段名）
}

func (jq *JoinQuery) resolveJoinRelationConfig(primaryType reflect.Type, relationName string) (*joinRelationConfig, error) {
	// 首先尝试从Relations()方法获取
	rel, err := getRelationConfig(primaryType, relationName)
	if err == nil && rel != nil {
		relType := getModelType(rel.Model)
		return &joinRelationConfig{
			name:         relationName,
			modelType:    relType,
			relationType: rel.Type,
			foreignKey:   rel.ForeignKey,
			primaryKey:   rel.PrimaryKey,
		}, nil
	}

	// 尝试从结构体字段推断
	return jq.inferRelationConfig(primaryType, relationName)
}

func (jq *JoinQuery) inferRelationConfig(primaryType reflect.Type, relationName string) (*joinRelationConfig, error) {
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

	fk := primaryType.Name() + "ID"
	pk := "ID"

	if relTypeStr == BelongsTo {
		fk = relationName + "ID"
	}

	return &joinRelationConfig{
		name:         relationName,
		modelType:    relType,
		relationType: relTypeStr,
		foreignKey:   fk,
		primaryKey:   pk,
	}, nil
}

func (jq *JoinQuery) buildJoinCondition(primaryType reflect.Type, config *joinRelationConfig) buildingsql.CondBuilder {
	mainTable := getTableNameFromType(primaryType)
	joinTable := getTableNameFromType(config.modelType)

	pk := config.primaryKey
	if pk == "" {
		pk = "id"
	}
	fk := config.foreignKey
	if fk == "" {
		fk = primaryType.Name() + "ID"
	}

	// 根据关联类型确定JOIN条件
	switch config.relationType {
	case BelongsTo:
		// belongs_to: 主表有外键
		mainFK := toDBFieldName(fk)
		joinPK := toDBFieldName(pk)
		return buildingsql.Compare(
			buildingsql.Field(mainTable, mainFK),
			"=",
			buildingsql.Field(joinTable, joinPK),
		)
	default:
		// has_one, has_many: 关联表有外键
		mainPK := toDBFieldName(pk)
		joinFK := toDBFieldName(primaryType.Name() + "ID")
		return buildingsql.Compare(
			buildingsql.Field(mainTable, mainPK),
			"=",
			buildingsql.Field(joinTable, joinFK),
		)
	}
}

// JoinResultSet JOIN查询结果集
type JoinResultSet struct {
	records   []map[string]any
	modelType reflect.Type
	joins     []joinDef
}

// Records 返回原始记录
func (jrs *JoinResultSet) Records() []map[string]any {
	return jrs.records
}

// Len 返回记录数
func (jrs *JoinResultSet) Len() int {
	return len(jrs.records)
}

// IsEmpty 是否为空
func (jrs *JoinResultSet) IsEmpty() bool {
	return len(jrs.records) == 0
}

// ==================== 便捷函数（使用CurrentDB）====================

// Join 开始一个JOIN查询（使用CurrentDB）
func Join(model any) *JoinQuery {
	return CurrentDB().Join(model)
}

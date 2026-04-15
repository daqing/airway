package repo

import (
	"reflect"
)

// RelationType 定义关联类型
type RelationType string

const (
	HasOneRelation    RelationType = "has_one"
	HasManyRelation   RelationType = "has_many"
	BelongsTo         RelationType = "belongs_to"
)

// Relation 定义模型关联配置
type Relation struct {
	Type       RelationType // 关联类型
	Name       string       // 关联名称
	Model      any          // 关联模型类型（实例或指针）
	ForeignKey string       // 外键字段名（Go字段名）
	PrimaryKey string       // 主键字段名（Go字段名，默认ID）
}

// Relational 接口，模型可以实现此接口来定义关联
type Relational interface {
	Relations() map[string]Relation
}

// HasOne 创建HasOne关联配置
// model: 关联模型实例
// foreignKey: 关联表中的外键字段名（如 "UserID"）
// primaryKey: 主表中的主键字段名（默认为 "ID"）
func HasOne(model any, foreignKey string, primaryKey ...string) Relation {
	pk := "ID"
	if len(primaryKey) > 0 {
		pk = primaryKey[0]
	}
	return Relation{
		Type:       HasOneRelation,
		Model:      model,
		ForeignKey: foreignKey,
		PrimaryKey: pk,
	}
}

// HasMany 创建HasMany关联配置
// model: 关联模型实例
// foreignKey: 关联表中的外键字段名（如 "UserID"）
// primaryKey: 主表中的主键字段名（默认为 "ID"）
func HasMany(model any, foreignKey string, primaryKey ...string) Relation {
	pk := "ID"
	if len(primaryKey) > 0 {
		pk = primaryKey[0]
	}
	return Relation{
		Type:       HasManyRelation,
		Model:      model,
		ForeignKey: foreignKey,
		PrimaryKey: pk,
	}
}

// NewBelongsTo 创建BelongsTo关联配置
// model: 关联模型实例
// foreignKey: 主表中的外键字段名（如 "AuthorID"）
// primaryKey: 关联表中的主键字段名（默认为 "ID"）
func NewBelongsTo(model any, foreignKey string, primaryKey ...string) Relation {
	pk := "ID"
	if len(primaryKey) > 0 {
		pk = primaryKey[0]
	}
	return Relation{
		Type:       BelongsTo,
		Model:      model,
		ForeignKey: foreignKey,
		PrimaryKey: pk,
	}
}

// getRelationConfig 从模型类型和关系名获取配置
func getRelationConfig(modelType reflect.Type, relationName string) (*Relation, error) {
	// 尝试从模型的 Relations 方法获取配置
	if relational, ok := reflect.New(modelType).Interface().(Relational); ok {
		relations := relational.Relations()
		if rel, exists := relations[relationName]; exists {
			return &rel, nil
		}
	}
	return nil, nil // 没有找到配置
}

// getModelType 获取模型的reflect.Type
func getModelType(model any) reflect.Type {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

package repo

import (
	buildingsql "github.com/daqing/airway/lib/sql"
)

// ==================== Create ====================

// CreateFrom 从map创建记录（使用CurrentDB）
func CreateFrom[T buildingsql.Table](vals buildingsql.H) (*T, error) {
	var t T
	b := buildingsql.Create(t, vals)
	return Insert[T](CurrentDB(), b)
}

// ==================== Find ====================

// FindBy 根据条件查询多条记录（使用CurrentDB）
func FindBy[T buildingsql.Table](vals buildingsql.H) ([]*T, error) {
	var t T
	b := buildingsql.FindByCond(t, buildingsql.MatchTable(t, vals))
	return Find[T](CurrentDB(), b)
}

// FindOneBy 根据条件查询单条记录（使用CurrentDB）
func FindOneBy[T buildingsql.Table](vals buildingsql.H) (*T, error) {
	var t T
	b := buildingsql.FindByCond(t, buildingsql.MatchTable(t, vals))
	return FindOne[T](CurrentDB(), b)
}

// FindByID 根据ID查询（使用CurrentDB）
func FindByID[T buildingsql.Table](id buildingsql.IdType) (*T, error) {
	var t T
	b := buildingsql.FindByCond(t, buildingsql.FieldEq(buildingsql.FieldFor(t, "id"), id))
	return FindOne[T](CurrentDB(), b)
}

// FindAll 查询所有记录（使用CurrentDB）
func FindAll[T buildingsql.Table]() ([]*T, error) {
	var t T
	b := buildingsql.All(t)
	return Find[T](CurrentDB(), b)
}

// ==================== Update ====================

// UpdateWhere 根据条件更新（使用CurrentDB）
func UpdateWhere[T buildingsql.Table](vals buildingsql.H, cond buildingsql.CondBuilder) error {
	var t T
	b := buildingsql.UpdateTable(buildingsql.TableFor(t)).Set(vals).Where(cond)
	return Update(CurrentDB(), b)
}

// UpdateByID 根据ID更新（使用CurrentDB）
func UpdateByID[T buildingsql.Table](id buildingsql.IdType, vals buildingsql.H) error {
	var t T
	b := buildingsql.UpdateTable(buildingsql.TableFor(t)).Set(vals).
		Where(buildingsql.FieldEq(buildingsql.FieldFor(t, "id"), id))
	return Update(CurrentDB(), b)
}

// UpdateEvery 更新所有记录（使用CurrentDB）
func UpdateEvery[T buildingsql.Table](vals buildingsql.H) error {
	var t T
	b := buildingsql.UpdateAll(t, vals)
	return Update(CurrentDB(), b)
}

// ==================== Delete ====================

// DeleteWhere 根据条件删除（使用CurrentDB）
func DeleteWhere[T buildingsql.Table](vals buildingsql.H) error {
	var t T
	b := buildingsql.DeleteFrom(buildingsql.TableFor(t)).Where(buildingsql.MatchTable(t, vals))
	return Delete(CurrentDB(), b)
}

// DeleteByID 根据ID删除（使用CurrentDB）
func DeleteByID[T buildingsql.Table](id buildingsql.IdType) error {
	return DeleteWhere[T](buildingsql.H{"id": id})
}

// DeleteEvery 删除所有记录（使用CurrentDB）
func DeleteEvery[T buildingsql.Table]() error {
	var t T
	b := buildingsql.DeleteFrom(buildingsql.TableFor(t))
	return Delete(CurrentDB(), b)
}

// ==================== Exists & Count ====================

// ExistsWhere 检查记录是否存在（使用CurrentDB）
func ExistsWhere[T buildingsql.Table](vals buildingsql.H) (bool, error) {
	var t T
	b := buildingsql.Exists(t, vals)
	return Exists(CurrentDB(), b)
}

// CountWhere 统计记录数（使用CurrentDB）
func CountWhere[T buildingsql.Table](cond buildingsql.H) (int64, error) {
	var t T
	b := buildingsql.SelectColumns("count(*)").FromTable(buildingsql.TableFor(t)).
		Where(buildingsql.MatchTable(t, cond))
	return Count(CurrentDB(), b)
}

// CountEvery 统计所有记录数（使用CurrentDB）
func CountEvery[T buildingsql.Table]() (int64, error) {
	return CountWhere[T](buildingsql.H{})
}

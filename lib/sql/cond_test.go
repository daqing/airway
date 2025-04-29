package sql

import "testing"

func TestEmptyCond(t *testing.T) {
	e := EmptyCond{}
	sql, vals := e.ToSQL()

	if sql != "" {
		t.Errorf("Expected empty SQL string, got: %s", sql)
	}

	if len(vals) != 0 {
		t.Errorf("Expected empty values slice, got: %v", vals)
	}
}

func TestCondition(t *testing.T) {
	c := Condition{
		Key: "name",
		Op:  "=",
		Val: "John",
	}
	sql, vals := c.ToSQL()

	expectedSQL := "name = @name"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(vals) != 1 || vals["name"] != "John" {
		t.Errorf("Expected values: map[name:John], got: %v", vals)
	}
}

func TestAndCond(t *testing.T) {
	c1 := Condition{
		Key: "name",
		Op:  "=",
		Val: "John",
	}
	c2 := Condition{
		Key: "age",
		Op:  ">",
		Val: 30,
	}

	andCond := AndCond{Conds: []Condition{c1, c2}}
	sql, vals := andCond.ToSQL()

	expectedSQL := "name = @name AND age > @age"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(vals) != 2 || vals["name"] != "John" || vals["age"] != 30 {
		t.Errorf("Expected values: [John, 30], got: %v", vals)
	}
}

func TestOrCond(t *testing.T) {
	c1 := Condition{
		Key: "name",
		Op:  "=",
		Val: "John",
	}
	c2 := Condition{
		Key: "age",
		Op:  ">",
		Val: 30,
	}

	orCond := OrCond{Conds: []Condition{c1, c2}}
	sql, vals := orCond.ToSQL()

	expectedSQL := "name = @name OR age > @age"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(vals) != 2 || vals["name"] != "John" || vals["age"] != 30 {
		t.Errorf("Expected values: [John, 30], got: %v", vals)
	}
}

func TestConditionGroup(t *testing.T) {
	c1 := Condition{
		Key: "name",
		Op:  "=",
		Val: "John",
	}
	c2 := Condition{
		Key: "age",
		Op:  ">",
		Val: 30,
	}

	group := ConditionGroup{
		Left:  &c1,
		Op:    And,
		Right: &c2,
	}
	sql, vals := group.ToSQL()

	expectedSQL := "(name = @name AND age > @age)"
	if sql != expectedSQL {
		t.Errorf("Expected SQL: %s, got: %s", expectedSQL, sql)
	}

	if len(vals) != 2 || vals["name"] != "John" || vals["age"] != 30 {
		t.Errorf("Expected values: [John, 30], got: %v", vals)
	}
}

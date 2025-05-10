package sql

func Eq(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "=",
		Val: val,
	}
}

func NotEq(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "<>",
		Val: val,
	}
}

func Gt(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  ">",
		Val: val,
	}
}

func Gte(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  ">=",
		Val: val,
	}
}

func Lt(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "<",
		Val: val,
	}
}

func Lte(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "<=",
		Val: val,
	}
}

func Like(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "LIKE",
		Val: val,
	}
}

func NotLike(key string, val any) *Condition {
	return &Condition{
		Key: key,
		Op:  "NOT LIKE",
		Val: val,
	}
}

func HCond(cond H) *MapCond {
	return &MapCond{
		Cond: cond,
	}
}

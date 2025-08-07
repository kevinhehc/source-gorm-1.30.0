package clause

// Select select attrs when querying, updating, creating
type Select struct {
	Distinct   bool     // 使用使用 distinct 模式
	Columns    []Column // 是否 select 查询指定的列，如 select id,name
	Expression Expression
}

func (s Select) Name() string {
	return "SELECT"
}

func (s Select) Build(builder Builder) {
	// select  查询指定的列
	if len(s.Columns) > 0 {
		if s.Distinct {
			builder.WriteString("DISTINCT ")
		}

		// 将指定列追加到 sql 语句中
		for idx, column := range s.Columns {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(column)
		}
	} else {
		// 不查询指定列，则使用 select *
		builder.WriteByte('*')
	}
}

func (s Select) MergeClause(clause *Clause) {
	if s.Expression != nil {
		if s.Distinct {
			if expr, ok := s.Expression.(Expr); ok {
				expr.SQL = "DISTINCT " + expr.SQL
				clause.Expression = expr
				return
			}
		}

		clause.Expression = s.Expression
	} else {
		clause.Expression = s
	}
}

// CommaExpression represents a group of expressions separated by commas.
type CommaExpression struct {
	Exprs []Expression
}

func (comma CommaExpression) Build(builder Builder) {
	for idx, expr := range comma.Exprs {
		if idx > 0 {
			_, _ = builder.WriteString(", ")
		}
		expr.Build(builder)
	}
}

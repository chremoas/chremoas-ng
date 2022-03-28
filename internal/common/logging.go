package common

import (
	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"go.uber.org/zap"
)

func LogSQL(sp *sl.Span, query interface{}) {
	var (
		sqlStr string
		args   []interface{}
		err    error
	)

	switch query.(type) {
	case sq.SelectBuilder:
		sqlStr, args, err = query.(sq.SelectBuilder).ToSql()
	case sq.InsertBuilder:
		sqlStr, args, err = query.(sq.InsertBuilder).ToSql()
	case sq.UpdateBuilder:
		sqlStr, args, err = query.(sq.UpdateBuilder).ToSql()
	case sq.DeleteBuilder:
		sqlStr, args, err = query.(sq.DeleteBuilder).ToSql()
	}

	if err != nil {
		sp.Error("error getting sql", zap.Error(err))
	} else {
		sp.Debug("sql query", zap.String("query", sqlStr), zap.Any("args", args))
	}
}

package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ EmailEventsModel = (*customEmailEventsModel)(nil)

type (
	// EmailEventsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customEmailEventsModel.
	EmailEventsModel interface {
		emailEventsModel
		withSession(session sqlx.Session) EmailEventsModel
	}

	customEmailEventsModel struct {
		*defaultEmailEventsModel
	}
)

// NewEmailEventsModel returns a model for the database table.
func NewEmailEventsModel(conn sqlx.SqlConn) EmailEventsModel {
	return &customEmailEventsModel{
		defaultEmailEventsModel: newEmailEventsModel(conn),
	}
}

func (m *customEmailEventsModel) withSession(session sqlx.Session) EmailEventsModel {
	return NewEmailEventsModel(sqlx.NewSqlConnFromSession(session))
}

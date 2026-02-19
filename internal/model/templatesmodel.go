package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ TemplatesModel = (*customTemplatesModel)(nil)

type (
	// TemplatesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTemplatesModel.
	TemplatesModel interface {
		templatesModel
		withSession(session sqlx.Session) TemplatesModel
	}

	customTemplatesModel struct {
		*defaultTemplatesModel
	}
)

// NewTemplatesModel returns a model for the database table.
func NewTemplatesModel(conn sqlx.SqlConn) TemplatesModel {
	return &customTemplatesModel{
		defaultTemplatesModel: newTemplatesModel(conn),
	}
}

func (m *customTemplatesModel) withSession(session sqlx.Session) TemplatesModel {
	return NewTemplatesModel(sqlx.NewSqlConnFromSession(session))
}

package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ SmtpProvidersModel = (*customSmtpProvidersModel)(nil)

type (
	// SmtpProvidersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSmtpProvidersModel.
	SmtpProvidersModel interface {
		smtpProvidersModel
		withSession(session sqlx.Session) SmtpProvidersModel
	}

	customSmtpProvidersModel struct {
		*defaultSmtpProvidersModel
	}
)

// NewSmtpProvidersModel returns a model for the database table.
func NewSmtpProvidersModel(conn sqlx.SqlConn) SmtpProvidersModel {
	return &customSmtpProvidersModel{
		defaultSmtpProvidersModel: newSmtpProvidersModel(conn),
	}
}

func (m *customSmtpProvidersModel) withSession(session sqlx.Session) SmtpProvidersModel {
	return NewSmtpProvidersModel(sqlx.NewSqlConnFromSession(session))
}

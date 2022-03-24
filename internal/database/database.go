package database

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	sl "github.com/bhechinger/spiffylogger"
	"github.com/jmoiron/sqlx"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func New(ctx context.Context) (*sq.StatementBuilderType, error) {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var err error

	// ignoredRoles = viper.GetStringSlice("bot.ignoredRoles")

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.database"),
	)

	ldb, err := sqlx.Connect(viper.GetString("database.driver"), dsn)
	if err != nil {
		sp.Error("Error connecting to DB", zap.Error(err))
		return nil, err
	}

	err = ldb.Ping()
	if err != nil {
		sp.Error("Error pinging DB", zap.Error(err))
		return nil, err
	}

	dbCache := sq.NewStmtCache(ldb)
	db := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(dbCache)

	// Ensure required permissions exist in the database
	var (
		requiredPermissions = map[string]string{
			"role_admins":   "Role Admins",
			"sig_admins":    "SIG Admins",
			"server_admins": "Server Admins",
		}
		id int
	)

	for k, v := range requiredPermissions {
		err = db.Select("id").
			From("permissions").
			Where(sq.Eq{"name": k}).
			Scan(&id)

		switch err {
		case nil:
			sp.Info("permission found", zap.String("permission", k), zap.Int("id", id))
		case sql.ErrNoRows:
			sp.Info("permission NOT found, creating", zap.String("permission", k))
			err = db.Insert("permissions").
				Columns("name", "description").
				Values(k, v).
				Suffix("RETURNING \"id\"").
				Scan(&id)
			if err != nil {
				sp.Error("Error inserting permissions", zap.Error(err))
				return nil, err
			}
		default:
			sp.Error("Error checking permissions", zap.Error(err))
			return nil, err
		}
	}

	return &db, nil
}

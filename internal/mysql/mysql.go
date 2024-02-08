package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

type ConnectionSetting struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func Gen(c ConnectionSetting) error {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", c.User, c.Password, c.Host, c.Port, c.Database))
	if err != nil {
		return err
	}
	defer db.Close()

	return nil
}

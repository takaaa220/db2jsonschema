package cmd

import (
	"github.com/spf13/cobra"
	"github.com/takaaa220/db2jsonschema/internal/mysql"
)

var (
	host            *string
	port            *int
	user            *string
	password        *string
	database        *string
	dateTimePattern *string
)

// mysqlCmd represents the mysql command
var mysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "convert mysql schema to json schema",
	Run: func(cmd *cobra.Command, args []string) {
		err := mysql.Gen(
			mysql.ConnectionSetting{
				Host:     *host,
				Port:     *port,
				User:     *user,
				Password: *password,
				Database: *database,
			},
			mysql.GenSetting{
				DateTimePattern: *dateTimePattern,
			},
		)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	host = mysqlCmd.Flags().StringP("host", "H", "localhost", "mysql host")
	port = mysqlCmd.Flags().IntP("port", "P", 3306, "mysql port")
	user = mysqlCmd.Flags().StringP("user", "u", "root", "mysql	user")
	password = mysqlCmd.Flags().StringP("password", "p", "", "mysql password")
	database = mysqlCmd.Flags().StringP("database", "d", "information_schema", "mysql database")
	dateTimePattern = mysqlCmd.Flags().StringP("datetime-pattern", "", "[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}", "datetime pattern")

	rootCmd.AddCommand(mysqlCmd)
}

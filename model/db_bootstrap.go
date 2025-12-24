package model

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func ensureDatabaseExistsFromEnv(envName string) error {
	dsn := os.Getenv(envName)
	if dsn == "" {
		return nil
	}
	if strings.HasPrefix(dsn, "local") {
		return nil
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return ensurePostgresDatabaseExists(dsn, envName)
	}
	return ensureMySQLDatabaseExists(dsn, envName)
}

func ensureMySQLDatabaseExists(dsn string, envName string) error {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("解析 %s 失败: %w", envName, err)
	}
	dbName := strings.TrimSpace(cfg.DBName)
	if dbName == "" {
		return nil
	}
	cfg.DBName = ""

	adminDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("连接 MySQL 失败: %w", err)
	}
	defer adminDB.Close()

	if err := adminDB.Ping(); err != nil {
		return fmt.Errorf("MySQL Ping 失败: %w", err)
	}

	exists, err := mysqlDatabaseExists(adminDB, dbName)
	if err != nil {
		return fmt.Errorf("检查 MySQL 数据库是否存在失败: %w", err)
	}
	if exists {
		return nil
	}

	common.SysLog(fmt.Sprintf("%s 数据库不存在，正在创建：%s", envName, dbName))
	if _, err := adminDB.Exec("CREATE DATABASE " + quoteMySQLIdentifier(dbName)); err != nil {
		recheck, reErr := mysqlDatabaseExists(adminDB, dbName)
		if reErr == nil && recheck {
			return nil
		}
		return fmt.Errorf("创建 MySQL 数据库失败: %w", err)
	}
	common.SysLog(fmt.Sprintf("%s 数据库创建完成：%s", envName, dbName))
	return nil
}

func mysqlDatabaseExists(db *sql.DB, dbName string) (bool, error) {
	var name string
	err := db.QueryRow("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", dbName).Scan(&name)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

func quoteMySQLIdentifier(name string) string {
	escaped := strings.ReplaceAll(name, "`", "``")
	return "`" + escaped + "`"
}

func ensurePostgresDatabaseExists(dsn string, envName string) error {
	parsed, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("解析 %s 失败: %w", envName, err)
	}
	dbName := strings.TrimPrefix(parsed.Path, "/")
	dbName = strings.TrimSpace(dbName)
	if dbName == "" {
		return nil
	}

	adminDB, err := openPostgresAdminDB(parsed)
	if err != nil {
		return fmt.Errorf("连接 PostgreSQL 管理库失败: %w", err)
	}
	defer adminDB.Close()

	exists, err := postgresDatabaseExists(adminDB, dbName)
	if err != nil {
		return fmt.Errorf("检查 PostgreSQL 数据库是否存在失败: %w", err)
	}
	if exists {
		return nil
	}

	common.SysLog(fmt.Sprintf("%s 数据库不存在，正在创建：%s", envName, dbName))
	if _, err := adminDB.Exec("CREATE DATABASE " + quotePostgresIdentifier(dbName)); err != nil {
		recheck, reErr := postgresDatabaseExists(adminDB, dbName)
		if reErr == nil && recheck {
			return nil
		}
		return fmt.Errorf("创建 PostgreSQL 数据库失败: %w", err)
	}
	common.SysLog(fmt.Sprintf("%s 数据库创建完成：%s", envName, dbName))
	return nil
}

func openPostgresAdminDB(baseURL *url.URL) (*sql.DB, error) {
	candidates := []string{"postgres", "template1"}
	var lastErr error
	for _, dbName := range candidates {
		adminURL := *baseURL
		adminURL.Path = "/" + dbName
		adminDB, err := sql.Open("pgx", adminURL.String())
		if err != nil {
			lastErr = err
			continue
		}
		if err := adminDB.Ping(); err != nil {
			adminDB.Close()
			lastErr = err
			continue
		}
		return adminDB, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("无法连接到 PostgreSQL 管理库")
	}
	return nil, lastErr
}

func postgresDatabaseExists(db *sql.DB, dbName string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)", dbName).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func quotePostgresIdentifier(name string) string {
	escaped := strings.ReplaceAll(name, "\"", "\"\"")
	return "\"" + escaped + "\""
}

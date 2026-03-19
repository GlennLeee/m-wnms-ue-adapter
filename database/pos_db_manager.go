package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/microsoft/go-mssqldb"
)

type PositionRdbConf struct {
	username string
	password string
	url      string
	port     string
	schema   string
	C        *sql.DB
	retryCnt int
}

func (p *PositionRdbConf) connection() error {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		time.Sleep(1000 * time.Millisecond)
	// 		p.retryCnt++
	// 		log.Println("[Connection Error] ", p.retryCnt, " Retry")
	// 		p.connection()
	// 	}
	// }()

	p.retryCnt = 0
	dataSource := fmt.Sprintf(
		"sqlserver://%s:%s@%s:%s?database=%s&encrypt=disable",
		p.username,
		p.password,
		p.url,
		p.port,
		p.schema,
	)
	conn, err := sql.Open("sqlserver", dataSource)

	if err != nil {
		log.Println("RDB Connection Open Error:", err)
		return err
	}

	// connection check
	if err = conn.Ping(); err != nil {
		conn.Close()
		log.Println("RDB Connection Ping Error:", err)
		return err
	}

	log.Println("DB Opened [", p.url, "]")
	p.C = conn
	return nil
}

func (p *PositionRdbConf) Initialize(url string, username string, password string, schema string, port string) error {
	p.username = username
	p.password = password
	p.schema = schema
	p.url = url
	p.port = port

	if p.url == "" {
		return fmt.Errorf("position db host is empty")
	}

	return p.connection()
}

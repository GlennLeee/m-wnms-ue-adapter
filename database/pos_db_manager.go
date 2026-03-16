package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
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

func (p *PositionRdbConf) connection() {
	defer func() {
		if err := recover(); err != nil {
			time.Sleep(1000 * time.Millisecond)
			p.retryCnt++
			log.Println("[Connection Error] ", p.retryCnt, " Retry")
			p.connection()
		}
	}()

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
		log.Fatal(err)
	}

	// connection check
	if err = conn.Ping(); err != nil {
		conn.Close()
		panic("RDB Connection Error")
	}

	log.Println("DB Opened [", p.url, "]")
	p.C = conn
}

func (p *PositionRdbConf) Initialize(url string, username string, password string, schema string, port string) {
	p.username = username
	p.password = password
	p.schema = schema
	p.url = url
	p.port = port

	p.connection()
}

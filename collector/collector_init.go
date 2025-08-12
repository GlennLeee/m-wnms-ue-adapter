package collector

import (
	"log"
	"regexp"
	"strings"

	"connectlabs.co.kr/wnms/wnms-ue-adapter/database"
)

type Data map[string]interface{}

var nMap = make(map[string]string)
var numberMap = make(map[string]string)
var nFormula = make(map[string]string)
var neededValue = make(map[string]string)

type Collector struct {
	db *database.RdbConf
}

func InitCollector(db *database.RdbConf) *Collector {
	c := &Collector{
		db: db,
	}

	c.loadKeyMap()
	c.loadFormula()

	return c
}

func (c *Collector) loadFormula() {

	qRes, _ := c.db.C.Query("SELECT name, data FROM p5g_collect_formula WHERE vendor='nokia'")
	mc := regexp.MustCompile("[+/*-/(/)]")
	chkNm := regexp.MustCompile(`^[+-]?\d*(\.?\d*)$`)

	defer qRes.Close() //반드시 닫는다 (지연하여 닫기)
	var k, v string
	for qRes.Next() {
		qRes.Scan(&k, &v)
		tmpFormula := strings.ToUpper(v)

		trimmedFormula := strings.ReplaceAll(tmpFormula, " ", "")
		t := mc.Split(trimmedFormula, -1)

		for _, target := range t {
			if target == "" {
				continue
			}
			if convTarget, ok := nMap[target]; ok {
				trimmedFormula = strings.ReplaceAll(trimmedFormula, target, convTarget)
				neededValue[convTarget] = target
			} else {
				if !chkNm.MatchString(target) {
					log.Println(target, "of", k, "value is not available")
				} else {
					numberMap[target] = target
				}
			}
		}
		nFormula[k] = trimmedFormula
	}
}

func (c *Collector) loadKeyMap() {

	qRes, _ := c.db.C.Query("SELECT * FROM p5g_collect_n_mapper")

	defer qRes.Close() //반드시 닫는다 (지연하여 닫기)
	var k, v string
	for qRes.Next() {
		qRes.Scan(&k, &v)
		nMap[strings.ToUpper(k)] = strings.ToUpper(v)
	}
}

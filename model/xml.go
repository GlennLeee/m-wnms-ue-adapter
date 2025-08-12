package model

type MeasCollecFile struct {
	MeasData   MeasData   `xml:"measData"`
	FileHeader FileHeader `xml:"fileHeader"`
	FileFooter FileFooter `xml:"fileFooter"`
}

type MeasData struct {
	MeasInfo []MeasInfo `xml:"measInfo"`
}

type MeasInfo struct {
	MeasInfoId string     `xml:"measInfoId,attr"`
	MeasTypes  string     `xml:"measTypes"`
	GranPeriod GranPeriod `xml:"granPeriod"`
	RepPeriod  RepPeriod  `xml:"repPeriod"`
	MeasValue  MeasValue  `xml:"measValue"`
}

type MeasValue struct {
	MeasObjLdn  string `xml:"measObjLdn,attr"`
	MeasResults string `xml:"measResults"`
}

type FileHeader struct {
	MeasCollec MeasCollec `xml:"measCollec"`
}

type FileFooter struct {
	MeasCollec MeasCollec `xml:"measCollec"`
}

type MeasCollec struct {
	BeginTime string `xml:"beginTime,attr"`
	EndTime   string `xml:"endTime,attr"`
}

type GranPeriod struct {
	Duration string `xml:"duration,attr"`
	EndTime  string `xml:"endTime,attr"`
}

type RepPeriod struct {
	Duration string `xml:"duration,attr"`
}

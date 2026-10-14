package discount

var codes = map[string]int64{"WELCOME10": 10}

func Percent(code string) int64 {
	return codes[code]
}

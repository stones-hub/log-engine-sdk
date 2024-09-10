package k3

import "regexp"

func parseTime(input []byte) string {
	var re = regexp.MustCompile(`"((\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})(?:\.(\d{3}))\d*)(Z|[\+-]\d{2}:\d{2})"`)
	var substitution = "\"$2 $3.$4\""

	for re.Match(input) {
		input = re.ReplaceAll(input, []byte(substitution))
	}
	return string(input)
}

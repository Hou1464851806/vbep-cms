package utils

import "regexp"

var hexRegex = regexp.MustCompile("^[0-9a-fA-F-]+$")

func IsHex(s string) bool {
    return hexRegex.MatchString(s)
}

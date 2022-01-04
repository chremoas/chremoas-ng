package queue

import (
	"regexp"
)

func sanitizeURI(uri string) string {
	re := regexp.MustCompile(`(amqp:\/\/[^:]*)(:.*@)(.*?$)`)

	return re.ReplaceAllString(uri, "$1:REDACTED@$3")
}

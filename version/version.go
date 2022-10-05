package version

import "fmt"

const Name = "Tart"

// injected by build arg
var (
	Tag      = "<unknown-tag>"
	Revision = "00000000"
)

// FullName initialized by init
var FullName string

func init() {
	FullName = fmt.Sprintf("%s %s(%s)", Name, Tag, Revision)
}

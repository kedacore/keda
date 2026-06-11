package contributors

import (
	_ "embed"
	"strings"
)

//go:generate bash -c "git log \"--pretty=%an <%ae>\" | sort -u > list"
//go:embed list
var source string

var List []string = strings.Split(source, "\n")

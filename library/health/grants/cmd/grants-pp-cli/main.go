// grants-pp-cli — nyitott kutatási pályázatok kulcs nélkül / open research grants, keyless.
// Források / sources: Grants.gov (nyitott kiírások), NIH RePORTER + NSF (megítélt grantek).
package main

import (
	"os"

	"github.com/mvanhorn/printing-press-library/library/health/grants/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}

// +build !windows

package commands

import "os"

//Writer represents a unix writer
var Writer = os.Stdout

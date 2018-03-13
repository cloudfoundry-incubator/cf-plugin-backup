package termuiprogressbar

import (
	"github.com/cheggaaa/pb"
)

type progressbar struct {
	progressbar *pb.ProgressBar
	visible     bool
}

// NewProgressBar creates a new progressbar
func NewProgressBar(total int, visible bool) *progressbar {
	return &progressbar{
		progressbar: pb.New64(int64(total)),
		visible:     visible,
	}
}

// Start starts the progressbar
func (pb *progressbar) Start() {
	if pb.visible {
		pb.progressbar.Start()
	}
}

// Increment increments the progressbar
func (pb *progressbar) Increment() {
	if pb.visible {
		pb.progressbar.Increment()
	}
}

// FinishPrint gets the progressbar to 100% and prints out a message
func (pb *progressbar) FinishPrint(str string) {
	if pb.visible {
		pb.progressbar.FinishPrint(str)
	}
}

package node

import "github.com/tez-capital/tezbake/ami"

func (app *Node) Start(args ...string) (int, error) {
	return ami.StartApp(app.GetPath(), args...)
}

func (app *Node) Stop(args ...string) (int, error) {
	return ami.StopApp(app.GetPath(), args...)
}

func (app *Node) Remove(all bool, args ...string) (int, error) {
	return ami.RemoveApp(app.GetPath(), all, args...)
}

func (app *Node) Execute(args ...string) (int, error) {
	return ami.Execute(app.GetPath(), args...)
}

func (app *Node) ExecuteGetOutput(args ...string) (string, int, error) {
	return ami.ExecuteGetOutput(app.GetPath(), args...)
}

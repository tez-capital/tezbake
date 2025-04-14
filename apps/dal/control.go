package dal

import "github.com/tez-capital/tezbake/ami"

func (app *DalNode) Start(args ...string) (int, error) {
	return ami.StartApp(app.GetPath(), args...)
}

func (app *DalNode) Stop(args ...string) (int, error) {
	return ami.StopApp(app.GetPath(), args...)
}

func (app *DalNode) Remove(all bool, args ...string) (int, error) {
	return ami.RemoveApp(app.GetPath(), all, args...)
}

func (app *DalNode) Execute(args ...string) (int, error) {
	return ami.Execute(app.GetPath(), args...)
}

func (app *DalNode) ExecuteGetOutput(args ...string) (string, int, error) {
	return ami.ExecuteGetOutput(app.GetPath(), args...)
}

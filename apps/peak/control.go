package peak

import "github.com/tez-capital/tezbake/ami"

func (app *Peak) Start(args ...string) (int, error) {
	return ami.StartApp(app.GetPath(), args...)
}

func (app *Peak) Stop(args ...string) (int, error) {
	return ami.StopApp(app.GetPath(), args...)
}

func (app *Peak) Remove(all bool, args ...string) (int, error) {
	return ami.RemoveApp(app.GetPath(), all, args...)
}

func (app *Peak) Execute(args ...string) (int, error) {
	return ami.Execute(app.GetPath(), args...)
}

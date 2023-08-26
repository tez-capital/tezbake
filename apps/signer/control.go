package signer

import "github.com/tez-capital/tezbake/ami"

func (app *Signer) Start(args ...string) (int, error) {
	return ami.StartApp(app.GetPath(), args...)
}

func (app *Signer) Stop(args ...string) (int, error) {
	return ami.StopApp(app.GetPath(), args...)
}

func (app *Signer) Remove(all bool, args ...string) (int, error) {
	return ami.RemoveApp(app.GetPath(), all, args...)
}

func (app *Signer) Execute(args ...string) (int, error) {
	return ami.Execute(app.GetPath(), args...)
}

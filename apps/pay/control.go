package pay

import "github.com/tez-capital/tezbake/ami"

func (app *Tezpay) Start(args ...string) (int, error) {
	return ami.StartApp(app.GetPath(), args...)
}

func (app *Tezpay) Stop(args ...string) (int, error) {
	return ami.StopApp(app.GetPath(), args...)
}

func (app *Tezpay) Remove(all bool, args ...string) (int, error) {
	return ami.RemoveApp(app.GetPath(), all, args...)
}

func (app *Tezpay) Execute(args ...string) (int, error) {
	return ami.Execute(app.GetPath(), args...)
}
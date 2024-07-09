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

func (app *Tezpay) ExecuteGetOutput(args ...string) (string, int, error) {
	return ami.ExecuteGetOutput(app.GetPath(), args...)
}

func (app *Tezpay) ExecuteWithOutputChannel(outputChannel chan<- string, args ...string) (int, error) {
	return ami.ExecuteWithOutputChannel(app.GetPath(), outputChannel, args...)
}

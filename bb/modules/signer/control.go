package bb_module_signer

import "alis.is/bb-cli/ami"

func (app *Signer) Start(args ...string) (int, error) {
	return ami.StartApp(app.GetPath(), args...)
}

func (app *Signer) Stop(args ...string) (int, error) {
	return ami.StopApp(app.GetPath(), args...)
}

func (app *Signer) Remove(all bool, args ...string) (int, error) {
	return ami.RemoveApp(app.GetPath(), all, args...)
}

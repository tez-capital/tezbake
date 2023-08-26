package bb_module_node

import "alis.is/bb-cli/ami"

func (app *Node) Start(args ...string) (int, error) {
	return ami.StartApp(app.GetPath(), args...)
}

func (app *Node) Stop(args ...string) (int, error) {
	return ami.StopApp(app.GetPath(), args...)
}

func (app *Node) Remove(all bool, args ...string) (int, error) {
	return ami.RemoveApp(app.GetPath(), all, args...)
}

package node

import "github.com/tez-capital/tezbake/apps/base"

func (app *Node) Pack(ctx *base.PackContext, output string) (int, error) {
	return base.DefaultPack(app, ctx, output)
}

func (app *Node) Unpack(source string) (int, error) {
	return base.DefaultUnpack(app, source)
}

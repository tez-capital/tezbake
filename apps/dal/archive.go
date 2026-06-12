package dal

import "github.com/tez-capital/tezbake/apps/base"

func (app *DalNode) Pack(ctx *base.PackContext, output string) (int, error) {
	return base.DefaultPack(app, ctx, output)
}

func (app *DalNode) Unpack(ctx *base.UnpackContext, source string) (int, error) {
	return base.DefaultUnpack(app, ctx, source)
}

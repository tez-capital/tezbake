package peak

import "github.com/tez-capital/tezbake/apps/base"

func (app *Peak) Pack(ctx *base.PackContext, output string) (int, error) {
	return base.DefaultPack(app, ctx, output)
}

func (app *Peak) Unpack(ctx *base.UnpackContext, source string) (int, error) {
	return base.DefaultUnpack(app, ctx, source)
}

package pay

import "github.com/tez-capital/tezbake/apps/base"

func (app *Tezpay) Pack(ctx *base.PackContext, output string) (int, error) {
	return base.DefaultPack(app, ctx, output)
}

func (app *Tezpay) Unpack(source string) (int, error) {
	return base.DefaultUnpack(app, source)
}

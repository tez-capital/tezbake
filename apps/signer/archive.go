package signer

import "github.com/tez-capital/tezbake/apps/base"

func (app *Signer) Pack(ctx *base.PackContext, output string) (int, error) {
	return base.DefaultPack(app, ctx, output)
}

func (app *Signer) Unpack(source string) (int, error) {
	return base.DefaultUnpack(app, source, "REMOTE_NODE", "REMOTE_SSH_PORT", "REMOTE_SSH_KEY", "REMOTE_RPC_ENDPOINT")
}

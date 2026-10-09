package agentexec

func requireContextPack(req Request) error {
	if req.ContextPack == nil {
		return ErrEffectGate
	}
	return nil
}

func contextPrefix(req Request) string {
	if req.ContextPack == nil {
		return ""
	}
	return req.ContextPack.Prefix
}

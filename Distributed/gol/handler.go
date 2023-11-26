package gol

func handler(p Params, c distributorChannels, keyPresses <-chan rune) {
	if p.Type == "d" {
		broker_distributor(p, c, keyPresses)
	}
}

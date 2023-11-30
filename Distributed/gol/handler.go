package gol

func handler(p Params, c distributorChannels, keyPresses <-chan rune) {
	// server_distribution(p, c, keyPresses)
	halo_distribution(p, c, keyPresses)
	// parallel_distributor(p, c, keyPresses)
	// if p.Type == "d" {
	// } else if p.Type == "p" {
	// 	parallel_distributor(p, c, keyPresses)
	// } else {
	// }
}

package assets

type Assets struct {
	LatticeApp         string
	TcpDropletReceiver string
}

func NewAssets() Assets {
	return Assets{
		LatticeApp:         "../assets/lattice-app.zip",
		TcpDropletReceiver: "../assets/tcp-droplet-receiver/",
	}
}

package assets

type Assets struct {
	TcpDropletReceiver string
}

func NewAssets() Assets {
	return Assets{
		TcpDropletReceiver: "../assets/tcp-droplet-receiver/",
	}
}

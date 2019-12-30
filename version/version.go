package version

var Version = "v0.0.1"

const Text = "Vince Server"

func Name() string {
	return Text + Version
}

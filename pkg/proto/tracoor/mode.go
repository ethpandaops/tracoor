package tracoor

type Mode string

const (
	ModeUnknown Mode = ""
	ModeAgent   Mode = "agent"
	ModeServer  Mode = "server"
)

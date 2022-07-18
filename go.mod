module termbot

go 1.18

require github.com/hinshun/vt10x v0.0.0-20220301184237-5011da428d02

require (
	github.com/gorilla/websocket v1.4.2 // indirect
	golang.org/x/crypto v0.0.0-20210421170649-83a5a9bb288b // indirect
)

require (
	github.com/bwmarrin/discordgo v0.25.0
	github.com/creack/pty v1.1.18
	golang.org/x/exp v0.0.0-20220713135740-79cabaa25d75
	golang.org/x/sys v0.0.0-20211019181941-9d821ace8654 // indirect
)

// replace termutil v0.0.0 => ./termutil
// require termutil v0.0.0

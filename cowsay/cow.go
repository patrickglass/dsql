package cowsay

const (
	cow = `---------------------------------------
         \   ^__^
          \  (oo)\_______
             (__)\       )\/\
                 ||----w |
                 ||     ||
	`
)

func Say(text string) string {
	return "\n" + text + "\n" + cow
}

package fmts

func init() {
	const lang = "ruby"

	register(&Fmt{
		Name: "rubocop",
		Errorformat: []string{
			`%A%f:%l:%c: %t: %m`,
			`%Z%p^%#`,
			`%C%.%#`,
			`%-G%.%#`,
		},
		Description: "A Ruby static code analyzer, based on the community Ruby style guide",
		URL:         "https://github.com/rubocop-hq/rubocop",
		Language:    lang,
	})

}

package utils

import "github.com/AlecAivazis/survey/v2"

// survey wrapper

func Ask(qs []*survey.Question, response interface{}, opts ...survey.AskOpt) error {
	opts = append(opts, survey.WithIcons(func(set *survey.IconSet) {
		set.Error.Format = "red+hb"
	}))
	return survey.Ask(qs, response, opts...)
}

func AskOne(p survey.Prompt, response interface{}, opts ...survey.AskOpt) error {
	opts = append(opts, survey.WithIcons(func(set *survey.IconSet) {
		set.Error.Format = "red+hb"
	}))
	return survey.AskOne(p, response, opts...)
}

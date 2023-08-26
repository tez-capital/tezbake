package ami

type Options struct {
	JsonLogFormat        bool   `json:"json_log_format"`
	LogLevel             string `json:"log_level"`
	DoNotCheckForLocator bool   `json:"do_not_check_for_locator"`
}

var (
	options = &Options{
		LogLevel:             "info",
		JsonLogFormat:        false,
		DoNotCheckForLocator: false,
	}
)

func GetOptions() Options {
	return *options
}

func SetOptions(newOptions Options) {
	options = &newOptions
}

func (opt *Options) ToAmiArgs() []string {
	amiArgs := make([]string, 0)
	if opt.JsonLogFormat {
		amiArgs = append(amiArgs, "--output-format=json")
	} else {
		amiArgs = append(amiArgs, "--output-format=standard")
	}
	amiArgs = append(amiArgs, "--log-level="+opt.LogLevel)
	return amiArgs
}

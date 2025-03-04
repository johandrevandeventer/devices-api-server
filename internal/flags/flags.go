package flags

// Default pattern to match files which trigger a build
const FilePattern = `(.+\.go|.+\.c)$`

var (
	// FlagInitConfig      bool
	FlagEnvironment string
	FlagDebugMode   bool
	FlagLogPrefix   bool
	FlagVerbose     bool
)

package font

const (
	// GoogleFontsAPI is the base URL for Google Fonts CSS API
	GoogleFontsAPI = "https://fonts.googleapis.com/css2"
	
	// DefaultFontFormat is the preferred font format for deck tools compatibility  
	DefaultFontFormat = "ttf"
	
	// RegistryFilename is the name of the font registry file
	RegistryFilename = "registry.json"
	
	// DefaultFontWeight is the standard font weight used when not specified
	DefaultFontWeight = 400
	
	// DefaultFontStyle is the standard font style used when not specified
	DefaultFontStyle = "normal"
)

// DefaultFonts contains the default Google Fonts used by deck visualization
var DefaultFonts = []string{
	"Roboto",
	"Open Sans",
	"Lato",
	"Montserrat",
	"Source Sans Pro",
	"Oswald",
	"Raleway",
	"Poppins",
	"Inter",
	"Ubuntu",
}
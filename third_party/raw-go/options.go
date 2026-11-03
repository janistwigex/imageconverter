package raw

// WhiteBalance selects how the white balance is computed when demosaicing a RAW image.
type WhiteBalance int

const (
	// WhiteBalanceCamera applies the camera's as-shot white balance. This is the default and yields colors closest to
	// what the camera intended.
	WhiteBalanceCamera WhiteBalance = iota
	// WhiteBalanceAuto computes an automatic white balance from the image data.
	WhiteBalanceAuto
	// WhiteBalanceNone applies no white balance, leaving LibRaw's fixed daylight balance.
	WhiteBalanceNone
)

// options holds the resolved decode parameters. Callers configure it through the Option functions.
type options struct {
	whiteBalance WhiteBalance
	autoBright   bool
	brightness   float64
}

// defaultOptions returns the decode defaults: the camera's white balance, auto-brightness enabled, and a neutral
// brightness multiplier. These produce a bright, natural rendering close to the camera's own JPEG.
func defaultOptions() options {
	return options{
		whiteBalance: WhiteBalanceCamera,
		autoBright:   true,
		brightness:   1.0,
	}
}

// Option configures how a RAW image is decoded.
type Option func(*options)

// WithWhiteBalance selects the white balance applied during decoding (default WhiteBalanceCamera).
func WithWhiteBalance(wb WhiteBalance) Option {
	return func(o *options) { o.whiteBalance = wb }
}

// WithAutoBright toggles LibRaw's auto-brightness histogram stretch (default true). Disable it to preserve the
// as-shot exposure.
func WithAutoBright(enabled bool) Option {
	return func(o *options) { o.autoBright = enabled }
}

// WithBrightness sets a brightness multiplier applied during decoding (default 1.0). Values above 1.0 brighten the
// image; values below 1.0 darken it.
func WithBrightness(multiplier float64) Option {
	return func(o *options) { o.brightness = multiplier }
}

// buildOptions folds the supplied Option functions over the defaults.
func buildOptions(opts []Option) options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

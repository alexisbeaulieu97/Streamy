package pipeline

// Settings captures global execution parameters for a pipeline.
type Settings struct {
	Parallel        int
	Timeout         int
	ContinueOnError bool
	DryRun          bool
	Verbose         bool
}

// Clone returns a copy of settings to avoid accidental mutations.
func (s Settings) Clone() Settings {
	return Settings{
		Parallel:        s.Parallel,
		Timeout:         s.Timeout,
		ContinueOnError: s.ContinueOnError,
		DryRun:          s.DryRun,
		Verbose:         s.Verbose,
	}
}

// ApplyDefaults ensures settings remain within supported ranges.
func (s Settings) ApplyDefaults() Settings {
	clone := s.Clone()
	if clone.Parallel <= 0 {
		clone.Parallel = 4
	}
	if clone.Timeout <= 0 {
		clone.Timeout = 300
	}
	return clone
}

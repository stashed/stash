package version

type Builder struct {
	Version
}

func (b *Builder) ResetMetadata() *Builder {
	b.metadata = ""
	return b
}

func (b *Builder) SetMetadata(md string) *Builder {
	b.metadata = md
	return b
}

func (b *Builder) ResetPrerelease() *Builder {
	b.pre = ""
	return b
}

func (b *Builder) SetPrerelease(id string) *Builder {
	b.pre = id
	return b
}

func (b *Builder) NextPatch() *Builder {
	if b.pre != "" {
		b.pre = ""
	} else {
		b.segments[2]++
	}
	return b
}

func (b *Builder) NextMinor() *Builder {
	if b.pre != "" {
		b.pre = ""
	} else {
		b.segments[1]++
		b.segments[2] = 0
	}
	return b
}

func (b *Builder) NextMajor() *Builder {
	if b.pre != "" {
		b.pre = ""
	} else {
		b.segments[0]++
		b.segments[1] = 0
		b.segments[2] = 0
	}
	return b
}

func (b *Builder) Done() *Version {
	return &b.Version
}

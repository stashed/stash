package version

type Mutator struct {
	*Version
}

func (m *Mutator) ResetMetadata() *Mutator {
	m.metadata = ""
	return m
}

func (m *Mutator) SetMetadata(md string) *Mutator {
	m.metadata = md
	return m
}

func (m *Mutator) ResetPrerelease() *Mutator {
	m.pre = ""
	return m
}

func (m *Mutator) SetPrerelease(id string) *Mutator {
	m.pre = id
	return m
}

func (m *Mutator) ResetPatch() *Mutator {
	m.pre = ""
	m.segments[2] = 0
	return m
}

func (m *Mutator) NextPatch() *Mutator {
	if m.pre != "" {
		m.pre = ""
	} else {
		m.segments[2]++
	}
	return m
}

func (m *Mutator) NextMinor() *Mutator {
	if m.pre != "" {
		m.pre = ""
	} else {
		m.segments[1]++
		m.segments[2] = 0
	}
	return m
}

func (m *Mutator) NextMajor() *Mutator {
	if m.pre != "" {
		m.pre = ""
	} else {
		m.segments[0]++
		m.segments[1] = 0
		m.segments[2] = 0
	}
	return m
}

func (m *Mutator) Done() *Version {
	return m.Version
}

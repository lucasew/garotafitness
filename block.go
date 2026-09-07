package garotafitness

// BlockKind is a FreeArc control-block type. Zero is invalid.
type BlockKind uint8

const (
	BlockInvalid BlockKind = iota
	BlockDescr
	BlockHeader
	BlockData
	BlockDir
	BlockFooter
	BlockRecovery
)

// FreeArc stores these as packed integers 0..5.
const (
	arcDescr    = 0
	arcHeader   = 1
	arcData     = 2
	arcDir      = 3
	arcFooter   = 4
	arcRecovery = 5
)

func blockKind(n int) BlockKind {
	switch n {
	case arcDescr:
		return BlockDescr
	case arcHeader:
		return BlockHeader
	case arcData:
		return BlockData
	case arcDir:
		return BlockDir
	case arcFooter:
		return BlockFooter
	case arcRecovery:
		return BlockRecovery
	default:
		return BlockInvalid
	}
}

func (k BlockKind) String() string {
	switch k {
	case BlockDescr:
		return "descr"
	case BlockHeader:
		return "header"
	case BlockData:
		return "data"
	case BlockDir:
		return "dir"
	case BlockFooter:
		return "footer"
	case BlockRecovery:
		return "recovery"
	default:
		return "invalid"
	}
}

package types

import (
	"fmt"
	tmtypes "github.com/tendermint/tendermint/types"
	"path"
	"strings"
)

type TemporaryHolder struct {
	files                []string
	tendermintGenesisDoc *tmtypes.GenesisDoc
}

func NewTemporaryHolder() *TemporaryHolder {
	return &TemporaryHolder{}
}

func (h *TemporaryHolder) AddTempFile(file string) {
	if len(file) < 1 {
		return
	}
	if !strings.HasPrefix(file, "/tmp/") {
		panic(fmt.Sprintf("temp file must be in '/tmp': %s", file))
	}
	_, name := path.Split(file)
	if !strings.Contains(name, ".tmp") {
		panic(fmt.Sprintf("temp file must contains part in '.tmp': %s", file))
	}
	h.files = append(h.files, file)
}

func (h *TemporaryHolder) CacheGenesisDoc(doc *tmtypes.GenesisDoc) {
	h.tendermintGenesisDoc = doc
}

func (h *TemporaryHolder) GetTempFiles() ([]string, bool) {
	return h.files, len(h.files) > 0
}

func (h *TemporaryHolder) GetCachedGenesisDoc() (*tmtypes.GenesisDoc, bool) {
	return h.tendermintGenesisDoc, h.tendermintGenesisDoc != nil
}

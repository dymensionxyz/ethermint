package types

import (
	cdb "github.com/cometbft/cometbft-db"
	tdb "github.com/cometbft/cometbft-db"
)

var _ cdb.DB = (*MemDB)(nil)

// MemDB is a wrapper of Tendermint/CometBFT DB that is backward-compatible with CometBFT chains pre-rename package.
//
// (eg: replace github.com/tendermint/tendermint => github.com/cometbft/cometbft v0.34.29)
type MemDB struct {
	tmDb tdb.DB
}

func WrapTendermintDB(tmDb tdb.DB) *MemDB {
	return &MemDB{tmDb: tmDb}
}

func (w *MemDB) AsCometBFT() cdb.DB {
	return w
}

func (w *MemDB) AsTendermint() tdb.DB {
	return w.tmDb
}

func (w *MemDB) Get(bytes []byte) ([]byte, error) {
	return w.tmDb.Get(bytes)
}

func (w *MemDB) Has(key []byte) (bool, error) {
	return w.tmDb.Has(key)
}

func (w *MemDB) Set(bytes []byte, bytes2 []byte) error {
	return w.tmDb.Set(bytes, bytes2)
}

func (w *MemDB) SetSync(bytes []byte, bytes2 []byte) error {
	return w.tmDb.SetSync(bytes, bytes2)
}

func (w *MemDB) Delete(bytes []byte) error {
	return w.tmDb.Delete(bytes)
}

func (w *MemDB) DeleteSync(bytes []byte) error {
	return w.tmDb.DeleteSync(bytes)
}

func (w *MemDB) Iterator(start, end []byte) (cdb.Iterator, error) {
	return w.tmDb.Iterator(start, end)
}

func (w *MemDB) ReverseIterator(start, end []byte) (cdb.Iterator, error) {
	return w.tmDb.ReverseIterator(start, end)
}

func (w *MemDB) Close() error {
	return w.tmDb.Close()
}

func (w *MemDB) NewBatch() cdb.Batch {
	return w.tmDb.NewBatch()
}

func (w *MemDB) Print() error {
	return w.tmDb.Print()
}

func (w *MemDB) Stats() map[string]string {
	return w.tmDb.Stats()
}
